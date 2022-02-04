// Copyright 2022 Styra Inc. All rights reserved.
// Use of this source code is governed by an Apache2
// license that can be found in the LICENSE file.

package playground

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/open-policy-agent/opa/plugins/bundle"
	"github.com/open-policy-agent/opa/sdk"
)

// opa stores the opa object to be used for obtaining decisions. This needs to
// be created using sdk.New().
var opa *sdk.OPA = nil
var opaContext context.Context

// rule stores the rule that should be queried from OPA, e.g. "/rules/allow".
var rulePath string

// allow stores the path to the boolean that will be true if the decision
// allowed the requested action. This is separate from the rule path, because
// we want DAS to get the full result object so it can correctly classify the
// decision as ALLOW/DENY.
var allowPath string

// bundleUpdateCounter tracks the number of times the bundle has been updated.
// This gets exposed to the API, so that the frontend can avoid polling for
// decisions if nothing has changed.
var bundleUpdateCounter int = 0

func SetOPA(newOPA *sdk.OPA, ctx context.Context) {
	opa = newOPA
	opaContext = ctx

	// NOTE: this variable carries state across calls to the below
	// callback.
	lastUpdate := time.Now()

	b := opa.Plugin("bundle").(*bundle.Plugin)
	b.Register("status_listener", func(status bundle.Status) {
		if status.LastSuccessfulDownload.After(lastUpdate) {
			lastUpdate = status.LastSuccessfulDownload
			bundleUpdateCounter++
		}
	})
}

func SetOPARule(newRule, newAllow string) {
	log.Printf("set rule path to '%s' and allow path to '%s'\n", newRule, newAllow)
	rulePath = newRule
	allowPath = newAllow
}

// walkResult is designed to extract values out of an OPA result object, which
// is a map of string keys to interface{} values, by subsequently trying each
// element in the path and returning the terminal one.
//
// If the requested path is not found, this function returns nil.
func walkResult(path []string, result map[string]interface{}) interface{} {
	if len(path) < 1 {
		return nil
	}

	rest := path[1:len(path)]

	// Ignore empty path elements we might get if the separator is
	// duplicated.
	if path[0] == "" {
		return walkResult(rest, result)
	}

	if len(path) == 1 {
		if value, ok := result[path[0]]; ok {
			return value
		}

		return nil // not OK, so we didn't find the key
	}

	if nested, ok := result[path[0]]; ok {
		if asmap, ok := nested.(map[string]interface{}); ok {
			return walkResult(rest, asmap)
		}

		// Cannot subscript further, since the nested object is not a map.
		return nil
	}

	return nil // not OK, so this path element wasn't found
}

func pretty(obj interface{}) string {
	// Pretty-print the result object as JOSN or panic.
	s, err := json.MarshalIndent(obj, "", "\t")
	if err != nil {
		panic(err)
	}
	return string(s)
}

// (response in JSON, allowed?, error)
//
// In principle, it would be better to use the entz abstractions in the sample
// package, however we already need to have direct access to the Go SDK so we
// can bind to bundle update events.
func response(input *FormInput) (interface{}, bool, error) {

	log.Printf("Asking OPA for a decision on %v\n", input)
	if opa == nil {
		log.Printf("OPA not configured, allowing operation")
		return struct {
			Msg string `json:"msg"`
		}{"OPA is not configured, all operations are allowed."}, true, nil
	}

	// TODO: what to do with Body?

	decOpts := sdk.DecisionOptions{
		Path: rulePath,
		Input: map[string]interface{}{
			"resource": input.Resource,
			"subject":  input.Subject,
			"action":   input.Action,
		},
	}

	result, err := opa.Decision(opaContext, decOpts)
	if err != nil {
		log.Printf("OPA error (denying request): %v\n", err)
		return struct {
			Message string `json:"msg"`
		}{fmt.Sprintf("OPA error: %v", err)}, false, err
	}

	asmap, ok := result.Result.(map[string]interface{})
	if !ok {
		panic(fmt.Sprintf("OPA result was not a map: %v\n", pretty(result.Result)))
	}

	allow := walkResult(strings.Split(allowPath, "/"), asmap)
	if allow == nil {
		panic(fmt.Sprintf("Expected OPA result to contain path '%s', but it did not. Result was: %v\n", allowPath, pretty(result)))
	}

	boolResult, ok := allow.(bool)
	if !ok {
		panic(fmt.Sprintf("Expected allow object to be boolean, are your rule path '%s' and allow path '%s' right? Result was: %v\n", rulePath, allowPath, allow))
	}

	log.Printf("OPA result: ID=%s, allowed=%v\n", result.ID, boolResult)

	return result, boolResult, nil
}
