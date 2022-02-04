package sample

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

// Entitlements represents an OPA input document, structured appropriately for
// use with the Entitlements system.
type EntitlementsInput struct {
	Action            string                 `json:"action"`
	Context           map[string]interface{} `json:"context"`
	Groups            []string               `json:"groups"`
	JWT               string                 `json:"jwt"`
	Resource          string                 `json:"resource"`
	ResourceAttribute map[string]string      `json:"resource-attributes"`
	Roles             []string               `json:"roles"`
	Subject           string                 `json:"subject"`
	SubjectAttributes map[string]string      `json:"subject-attributes"`
}

// EntitlementsResult represent an OPA result field created using an
// Entitlements policy.
type EntitlementsResult struct {
	Allowed bool                 `json:"allowed"`
	Entz    interface{}          `json:"entz"`
	Outcome *EntitlementsOutcome `json:"outcome"`
}

// EntitlementsRuleResult represents the output of a single Entitlements rule.
type EntitlementsRuleResult struct {
	Allowed bool        `json:"allowed"`
	Denied  bool        `json:"denied"`
	Entz    interface{} `json:"entz"`
	Message string      `json:"message"`
}

// EntitlementsOutcome represents the outcome field of an Entitlements result.
type EntitlementsOutcome struct {
	Allow        bool                      `json:"allow"`
	DecisionType string                    `json:"decision_type"`
	Enforced     []*EntitlementsRuleResult `json:"enforced"`
	Monitored    []*EntitlementsRuleResult `json:"monitored"`
	PolicyType   string                    `json:"policy_type"`
	Stacks       interface{}               `json:"stacks"`
	SystemType   string                    `json:"system_type"`
}

// Assert compliance with the http.Handler interface
var _ http.Handler = (*EntitlementsHandler)(nil)

// EntitlementsHandler is an http.Handler that checks all requests against an
// OPADecider, which is expected to return EntitlementsResult objects in it's
// result field.
//
// Because this is intended to be a simple example, we don't do any fancy
// authentication.
//
// The "User" is used as the subject field for entitlements requests.
//
// URL.Path is used as the resource field for entitlements requests.
//
// Method is used as the Action field for entitlements requests.
//
// All HTTP headers are passed into the Context field for entitlements
// requests in the "headers" sub-field.
type EntitlementsHandler struct {
	decider OPADecider
	handler http.Handler
}

// NewEntitlementsHandler instances a new EntitlementsHandler.
func NewEntitlementsHandler(decider OPADecider, handler http.Handler) *EntitlementsHandler {
	return &EntitlementsHandler{
		decider: decider,
		handler: handler,
	}
}

func jsonError(w http.ResponseWriter, message string, err error, code int) {
	var msg struct {
		Msg string `json:"msg"`
		Err string `json:"err"`
	}
	msg.Msg = message
	msg.Err = ""
	if err != nil {
		msg.Err = err.Error()
	}

	b, err := json.Marshal(msg)
	if err != nil {
		// should never happen
		panic(fmt.Sprintf("error while marshaling '%s': %v\n", msg, err))
	}

	http.Error(w, string(b), code)
}

// ServeHTTP implements http.Handler.ServeHTTP
func (h *EntitlementsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	entzContext := map[string]interface{}{}
	entzContext["headers"] = r.Header

	input := &EntitlementsInput{
		Action:   r.Method,
		Resource: r.URL.Path,
		Subject:  r.Header.Get("User"),
		Context:  entzContext,
	}

	decision, err := h.decider.Decision(input)
	if err != nil {
		jsonError(w, "failed to get decision for input", err, 500)
		return
	}

	resultJSON, err := json.Marshal(decision.Result)
	if err != nil {
		jsonError(w, "failed to marshal decision result", err, 500)
		return
	}

	result := &EntitlementsResult{}
	err = json.Unmarshal(resultJSON, result)
	if err != nil {
		jsonError(w, "failed to unmarshal decision result", err, 500)
		return
	}

	if !result.Allowed {
		log.Printf("%s %s %s: denied by decision %s\n", r.RemoteAddr, r.Method, r.URL.Path, decision.ID)
		jsonError(w, "action prohibited by Entitlements policy", nil, 403)
		return
	}

	log.Printf("%s %s %s: allowed by decision %s\n", r.RemoteAddr, r.Method, r.URL.Path, decision.ID)
	h.handler.ServeHTTP(w, r)
}
