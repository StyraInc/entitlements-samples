# Entitlements - Go Sample

This application demonstrates how to use Entitlements in two possible ways -
using OPA as a sidecar, or embedding OPA using the [OPA
SDK](https://pkg.go.dev/github.com/open-policy-agent/opa/sdk). To reduce
duplicated code, both methods of utilizing OPA are implemented, and are hidden
behind an abstraction layer.

The abstraction is implemented in [`opa.go`](./opa.go), and is a simple wrapper
interface, `SDKDecider` which presents a unified way of getting a decision from
OPA. Allow-all, deny-all, sidecar, and SDK based implementations are
implemented in the same file. In a production setting, it is suggested to
choose one specific way of working with OPA rather than using an abstraction
such as this one.
