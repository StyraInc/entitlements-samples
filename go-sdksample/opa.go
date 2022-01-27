// Copyright 2022 Styra Inc. All rights reserved.
// Use of this source code is governed by an Apache2
// license that can be found in the LICENSE file.

package sdksample

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

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

func SetOPA(newOPA *sdk.OPA, ctx context.Context) {
	opa = newOPA
	opaContext = ctx
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

// decision asks the OPA server for a decision on the given path, user, and
// method. If the operation is permitted, then it returns true, and otherwise
// false. If no OPA URL is configured, then it always returns true.
func decision(path []string, user, method string) (bool, error) {
	log.Printf("Asking OPA for a decision on path='%v' user='%s', method='%s'\n", path, user, method)
	if opa == nil {
		log.Printf("OPA not configured, allowing operation")
		return true, nil
	}

	decOpts := sdk.DecisionOptions{
		Path: rulePath,
		Input: map[string]interface{}{
			"resource": path,
			"subject":  user,
			"action":   method,
		},
	}

	result, err := opa.Decision(opaContext, decOpts)
	if err != nil {
		log.Printf("OPA error (denying request): %v\n", err)
		return false, err
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

	return boolResult, nil

}
