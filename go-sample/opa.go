// Copyright 2022 Styra Inc. All rights reserved.
// Use of this source code is governed by an Apache2
// license that can be found in the LICENSE file.

package sample

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/open-policy-agent/opa/sdk"
)

///////////////////////////////////////////////////////////////////////////////
//
// This file shows two different ways of interacting with OPA using Go. The
// first is to use the OPA SDK. In this case, OPA is directly embedded in the
// program as a Go library. The other method is to run OPA as a standalone
// process, and to interact with it via its REST API.
//
// Both methods are fully supported and offer equivalent functionality. Using
// the OPA SDK may improve performance by removing HTTP request overhead from
// obtaining decisions, however using a separate OPA binary allows OPA to be
// updated independently of the rest of the program, and allows multiple
// programs to use the same OPA instance. You should select the approach that
// works best for your specific use case.
//
///////////////////////////////////////////////////////////////////////////////

// OPADecision represents a decision (output document) from OPA.
type OPADecision struct {
	ID     string      `json:"ID"`
	Result interface{} `json:"result"`
}

// HTTPDecision represents a decision obtained via HTTP. The SDK has it's own
// decision format.
type HTTPDecision struct {
	Labels       map[string]string `json:"labels"`
	DecisionID   string            `json:"decision_id"`
	Path         string            `json:"path"`
	Input        interface{}       `json:"input"`
	Result       interface{}       `json:"result"`
	Timestamp    string            `json:"timestamp`
	Metrics      map[string]int    `json:"metrics"`
	AgentID      string            `json:"agent_id"`
	SystemID     string            `json:"system_id"`
	SystemType   string            `json:"system_type"`
	PolicyType   string            `json:"policy_type"`
	Received     string            `json:"received"`
	Allowed      map[string]bool   `json:"allowed"`
	DecisionType string            `json:"decision_type"`
	Columns      []string          `json:"columns"`
}

// OPADecider represents something capable of obtaining OPA decisions.
type OPADecider interface {

	// Decision should take an input, which must be JSON-serializeable,
	// and will be used as the input document for OPA.
	//
	// It returns the result object and an error, if any.
	Decision(input interface{}) (*OPADecision, error)
}

// Assert compliance with OPADecider
var _ OPADecider = (*SDKDecider)(nil)

type SDKDecider struct {
	opa  *sdk.OPA
	ctx  context.Context
	path string
}

// NewSDKDecider instances an OPADecider that uses the Go OPA SDK.
//
// opa should be obtained using sdk.New()
//
// path should be the rule path that is to be used when constructing
// sdk.DecisionOptions.
func NewSDKDecider(opa *sdk.OPA, ctx context.Context, path string) OPADecider {
	return &SDKDecider{
		opa:  opa,
		ctx:  ctx,
		path: path,
	}
}

// Decision implements OPADecider.Decision.
func (d *SDKDecider) Decision(input interface{}) (*OPADecision, error) {
	log.Printf("Asking OPA for a decision on input document %v\n", input)

	decOpts := sdk.DecisionOptions{
		Path:  d.path,
		Input: input,
	}

	result, err := d.opa.Decision(d.ctx, decOpts)
	if err != nil {
		log.Printf("OPA error (denying request): %v\n", err)
		return nil, err
	}

	return &OPADecision{ID: result.ID, Result: result.Result}, nil
}

// Assert compliance with OPADecider
var _ OPADecider = (*HTTPDecider)(nil)

type HTTPDecider struct {
	url string
}

// NewHTTPDecider instances an OPADecider that uses the OPA running as a
// sidecar, accessed using HTTP REST calls.
//
// The url should be the URL at which OPA should be queried.
func NewHTTPDecider(url string) OPADecider {
	return &HTTPDecider{
		url: url,
	}
}

// Decision implements OPADecider.Decision.
func (d *HTTPDecider) Decision(input interface{}) (*OPADecision, error) {
	log.Printf("Asking OPA for a decision on input document %v\n", input)

	// Prepare the data to be sent to the OPA server
	var reqStruct struct {
		Input interface{} `json:"input"`
	}
	reqStruct.Input = input
	reqData, err := json.Marshal(reqStruct)
	if err != nil {
		return nil, err
	}

	// Perform the PUT
	resp, err := http.Post(d.url, "application/json; charset=utf-8", bytes.NewBuffer(reqData))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Decode the response
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	decision := &HTTPDecision{}
	err = json.Unmarshal(bodyBytes, decision)
	if err != nil {
		return nil, err
	}

	return &OPADecision{ID: decision.DecisionID, Result: decision.Result}, nil
}

// Assert compliance with OPADecider
var _ OPADecider = (*DummyDecider)(nil)

type DummyDecider struct {
	decision *OPADecision
}

// NewDummyDecider instances an OPADecider that always returns the specified
// decision.
func NewDummyDecider(decision *OPADecision) OPADecider {
	return &DummyDecider{
		decision: decision,
	}
}

// Decision implement OPADecider.Decision.
func (d *DummyDecider) Decision(input interface{}) (*OPADecision, error) {
	return d.decision, nil
}
