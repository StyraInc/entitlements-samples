// Copyright 2022 Styra Inc. All rights reserved.
// Use of this source code is governed by an Apache2
// license that can be found in the LICENSE file.

package sdksample

import (
	"context"
	"fmt"
	"log"

	"github.com/open-policy-agent/opa/sdk"
)

// opa stores the opa object to be used for obtaining decisions. This needs to
// be created using sdk.New().
var opa *sdk.OPA = nil
var opaContext context.Context

// rule stores the rule that should be queried from OPA, e.g. "/rules/allow".
var rule string

func SetOPA(newOPA *sdk.OPA, ctx context.Context) {
	opa = newOPA
	opaContext = ctx
}

func SetOPARule(newRule string) {
	rule = newRule
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
		Path: rule,
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

	boolResult, ok := result.Result.(bool)
	if !ok {
		panic(fmt.Sprintf("Expected result to be boolean, is your rule path '%s' right? Result was: %v\n", rule, result))
	}

	log.Printf("OPA result: ID=%s, allowed=%v\n", result.ID, boolResult)

	return boolResult, nil

}
