// Copyright 2022 Styra Inc. All rights reserved.
// Use of this source code is governed by an Apache2
// license that can be found in the LICENSE file.

package main

import (
	"context"
	"fmt"
	"os"

	"github.com/alecthomas/kong"

	"github.com/styrainc/entitlements-samples/go-sdksample"

	"github.com/open-policy-agent/opa/logging"
	"github.com/open-policy-agent/opa/sdk"
)

var CLI struct {
	Storage string `name:"path" short:"p" type:"path" default:"./" help:"Directory where persistent data should be stored."`
	Port    int    `name:"port" short:"P" type:"int" default:8123 help:"Port where API should be served."`
	Config  string `name:"config" short:"o" type:"path" help:"Path to OPA configuration file. If omitted, OPA support will be disabled."`
	Rule    string `name:"rule" short:"r" default:"/main/main" type:"string" help:"OPA rule path"`
	Allow   string `name:"allow" short:"a" default:"outcome/allow" type:"string" help:"path within the OPA rule to extract the allow/deny decision"`
}

func main() {
	kong.Parse(&CLI)
	fmt.Printf("storage=%s\n", CLI.Storage)
	fmt.Printf("port=%d\n", CLI.Port)
	fmt.Printf("config=%s\n", CLI.Config)
	fmt.Printf("launching server ...\n")

	if CLI.Config != "" {
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

		err = sdksample.SetStorageDir(CLI.Storage)
		if err != nil {
			panic(err)
		}

		sdksample.SetOPA(opa, ctx)
		sdksample.SetOPARule(CLI.Rule, CLI.Allow)
	}

	sdksample.LoadFromDisk()
	sdksample.HandleRequests(CLI.Port)

}
