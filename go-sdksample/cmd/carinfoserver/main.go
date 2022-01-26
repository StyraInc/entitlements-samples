package main

import (
	"context"
	"fmt"
	"os"

	"github.com/alecthomas/kong"

	httpsample "github.com/styrainc/entitlements-samples/go-sdksample"

	"github.com/open-policy-agent/opa/sdk"
)

var CLI struct {
	Storage string `name:"path" short:"p" type:"path" default:"./" help:"Directory where persistent data should be stored."`
	Port    int    `name:"port" short:"P" type:"int" default:8123 help:"Port where API should be served."`
	Config  string `name:"config" short:"o" type:"path" help:"Path to OPA configuration file. If omitted, OPA support will be disabled."`
	Rule    string `name:"rule" short:"r" default:"/main/main/outcome/allow" type:"path" help:"OPA rule path"`
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
		})
		if err != nil {
			panic(err)
		}

		defer opa.Stop(ctx)

		err = httpsample.SetStorageDir(CLI.Storage)
		if err != nil {
			panic(err)
		}

		httpsample.SetOPA(opa, ctx)
		httpsample.SetOPARule(CLI.Rule)
	}

	httpsample.LoadFromDisk()
	httpsample.HandleRequests(CLI.Port)

}
