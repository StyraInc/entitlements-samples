// Copyright 2022 Styra Inc. All rights reserved.
// Use of this source code is governed by an Apache2
// license that can be found in the LICENSE file.

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/alecthomas/kong"

	"github.com/gorilla/mux"

	"github.com/styrainc/entitlements-samples/go-sample"
	"github.com/styrainc/entitlements-samples/go-sample/playground"

	"github.com/open-policy-agent/opa/logging"
	"github.com/open-policy-agent/opa/sdk"
)

var CLI struct {
	Storage    string `name:"path" short:"p" type:"path" default:"./" help:"Directory where persistent data should be stored."`
	Port       int    `name:"port" short:"P" type:"int" default:8123 help:"Port where API should be served."`
	Config     string `name:"config" short:"c" type:"path" help:"Path to OPA configuration file (sdk mode only)"`
	Rule       string `name:"rule" short:"r" default:"/main/main" type:"string" help:"OPA rule path (sdk mode only)"`
	Allow      string `name:"allow" short:"a" default:"outcome/allow" type:"string" help:"path within the OPA rule to extract the allow/deny decision"`
	OPA        string `name:"opa" short:"o" type:"string" help:"URL for the OPA server (http mode only)"`
	Mode       string `name:"mode" short:"m" type:"string" default:"sdk" help:"Mode in which to use OPA, choices are 'sdk', 'http', 'allow-all', 'deny-all'"`
	Playground bool   `name:"playground" short:"g" help:"Enable the /playground web UI. Only works in sdk mode."`
}

var dummyAllow string = `
{
  "ID": "ffffffff-ffff-ffff-ffff-ffffffffffff",
  "result": {
    "allowed": true,
    "entz": [],
    "outcome": {
      "allow": true,
      "decision_type": "ALLOWED",
      "enforced": [
        {
          "allowed": true,
          "entz": [],
          "message": "Request was matched"
        }
      ],
      "monitored": [],
      "policy_type": "rules",
      "stacks": {},
      "system_type": "template.entitlements:0.1"
    }
  }
}
`

var dummyDeny string = `
{
  "ID": "ffffffff-ffff-ffff-ffff-ffffffffffff",
  "result": {
    "allowed": false,
    "entz": [],
    "outcome": {
      "allow": true,
      "decision_type": "DENIED",
      "enforced": [
        {
          "denied": true,
          "entz": [],
          "message": "Request was matched"
        }
      ],
      "monitored": [],
      "policy_type": "rules",
      "stacks": {},
      "system_type": "template.entitlements:0.1"
    }
  }
}
`

func main() {
	kong.Parse(&CLI)

	var decider sample.OPADecider

	if CLI.Mode == "sdk" {

		if CLI.Config == "" {
			panic("config must be provided in sdk mode")
		}

		if CLI.Rule == "" {
			panic("rule must be provided in sdk mode")
		}

		ctx := context.Background()

		f, err := os.Open(CLI.Config)
		if err != nil {
			panic(err)
		}
		defer f.Close()

		// create a new OPA client with the config
		opa, err := sdk.New(ctx, sdk.Options{
			Config: f,

			// This is not suggested for production use, but is
			// nice for the sample as it allows seeing when OPA
			// has updated the policy bundle.
			Logger: logging.New(),
		})
		if err != nil {
			panic(err)
		}

		defer opa.Stop(ctx)

		decider = sample.NewSDKDecider(opa, ctx, CLI.Rule)

		if CLI.Playground {
			playground.SetOPARule("/main/main", "outcome/allow")
			playground.SetOPA(opa, ctx)
		}

	} else if CLI.Mode == "http" {

		decider = sample.NewHTTPDecider(CLI.OPA)

	} else if CLI.Mode == "allow-all" {

		decision := &sample.OPADecision{}
		err := json.Unmarshal([]byte(dummyAllow), decision)
		if err != nil {
			panic(err)
		}

		decider = sample.NewDummyDecider(decision)

	} else if CLI.Mode == "deny-all" {

		decision := &sample.OPADecision{}
		err := json.Unmarshal([]byte(dummyDeny), decision)
		if err != nil {
			panic(err)
		}

		decider = sample.NewDummyDecider(decision)

	} else {
		panic(fmt.Sprintf("mode '%s' is not one of sdk, http, allow-all, deny-all", CLI.Mode))
	}

	if CLI.Playground {

	}

	err := sample.SetStorageDir(CLI.Storage)
	if err != nil {
		panic(err)
	}

	sample.LoadFromDisk()

	r := mux.NewRouter().StrictSlash(false)
	carsRouter := r.PathPrefix("/cars")
	carsRouter.Handler(sample.NewEntitlementsHandler(decider, sample.GetAPIHandler()))
	//r.Handle("/cars{.*}", sample.NewEntitlementsHandler(decider, sample.GetAPIHandler()))

	if CLI.Playground {
		fmt.Printf("Enabling playground...\n")

		if CLI.Mode != "sdk" {
			panic("entz-playground can only be enabled in sdk mode")
		}

		playgroundRouter := r.PathPrefix("/")
		playgroundRouter.Handler(playground.GetAPIHandler())
		//r.Handle("/{.*}", playground.GetAPIHandler())
	}

	err = http.ListenAndServe(fmt.Sprintf(":%d", CLI.Port), r)
	if err != nil {
		panic(err)
	}

}
