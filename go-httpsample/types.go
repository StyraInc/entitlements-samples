// Copyright 2022 Styra Inc. All rights reserved.
// Use of this source code is governed by an Apache2
// license that can be found in the LICENSE file.

package httpsample

// Car represents information about a car on the lot.
type Car struct {
	// Make is the car's make, for example "Honda"
	Make string `json:"make"`

	// Model is the car's model, for example "Accord"
	Model string `json:"model"`

	// Year is the car's year of manufacture, for example 2017
	Year int `json:"year"`

	// Color is the color of the car's paint
	Color string `json:"color"`
}

// Status represents information about the status of the car.
type Status struct {
	// Sold is true if the car has already been sold.
	Sold bool `json:"sold"`

	// Ready is true if the car is ready to be sold.
	Ready bool `json:"ready"`

	// Price is the asking price for the car.
	Price float32 `json:"price"`
}

// PersistanceData represents the JSON data stored to disk by the persistence
// layer.
type PersistanceData struct {
	Cars     map[string]Car    `json:"cars"`
	Statuses map[string]Status `json:"statuses"`
}

// OPAInput represents the input data
type OPAInput struct {
	// Path represents the REST path components
	Path []string `json:"resource"`

	// User is the name of the user making the request (because this is an
	// example, we hand-wave any actual authentication).
	User string `json:"subject"`

	// Method is the HTTP method being used, e.g. GET, PUT, etc.
	Method string `json:"action"`
}

// OPADecision represents an entire decision from the OPA server.
type OPADecision struct {
	DecisionID string     `json:"decision_id"`
	Result     *OPAResult `json:"result"`
}

// OPAResult represents a decision returned by the OPA server.
type OPAResult struct {
	Allowed bool `json:"allowed"`
}
