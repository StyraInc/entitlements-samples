// Copyright 2022 Styra Inc. All rights reserved.
// Use of this source code is governed by an Apache2
// license that can be found in the LICENSE file.

package main

import (
	"fmt"

	"github.com/alecthomas/kong"

	"github.com/styrainc/entitlements-samples/go-httpsample"
)

var CLI struct {
	Storage string `name:"path" short:"p" type:"path" default:"./" help:"Directory where persistent data should be stored."`
	Port    int    `name:"port" short:"P" type:"int" default:8123 help:"Port where API should be served."`
	OPA     string `name:"opa" short:"o" type:"string" help:"To enable OPA support, supply the URL of the OPA server."`
}

func main() {
	kong.Parse(&CLI)
	fmt.Printf("storage=%s\n", CLI.Storage)
	fmt.Printf("port=%d\n", CLI.Port)
	fmt.Printf("OPA=%s\n", CLI.OPA)
	fmt.Printf("launching server ...\n")

	err := httpsample.SetStorageDir(CLI.Storage)
	if err != nil {
		panic(err)
	}

	httpsample.SetOPAURL(CLI.OPA)
	httpsample.LoadFromDisk()
	httpsample.HandleRequests(CLI.Port)

}
