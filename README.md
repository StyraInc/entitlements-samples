# Entitlements Samples

This repository contains examples of using the Entitlements feature of Styra
DAS in various languages and settings.

## How to Run

TODO: write instructions on how to run

Sample applications:

* `python-httpsample` - the [python sample using OPA as a sidecar over
  http](./python-httpsample).
* `go-httpsample` - the [go sample](./go-sample) running in OPA sidecar mode.
* `go-sdksample` - the [go sample](./go-sample) running OPA as an embedded Go
  library via the OPA SDK ("SDK mode").
* `entz-playground` - the [go sample](./go-sample) running in SDK mode, with
  the "entitlements playground" web interface enabled.

### Influential Environment Variables

You can use the following environment variables to influence the behavior of
the container via `-e`.

These environment variables **must** be set for the container to work:

* `DAS_TOKEN` - the token to be used for retrieving the OPA configuration from
  the DAS.
* `DAS_SYSTEM` - the DAS system ID we should pull rules from.
* `DAS_URL` - the full URL of the DAS instance we are to pull the system down
  from.
* `SAMPLE_APP` - which of the sample applications to run. See previous section
  for choices.

These environment variables may be specified, but have default values (shown in
parenthesis) if they are omitted:

* `API_PORT` (8123) - the port on which the API server for the sample should
  listen. When using the "entitlements playground", it is served on this port
  as well.
* `DOCS_PORT` (8080) - the port on which the API documentation server should
  listen.
* `TEST` - if this variable is set to any non-empty value, then the test suite
  is run against the chosen sample app. This variable is normally used only
  during sample development.

## Exercises

TODO: put in some helpful exercises for the user to run through.

```sh
curl -Ss localhost:8123/cars -H "user: alice"
```


## Tests

To run the test suite, simply issue a normal `docker run` command, but add `-e
TEST=YES`. The test suite will automatically run against the selected
`SAMPLE_APP`.

## How Does This Work?

The container image is built up from a plain Ubuntu base via the
[`Dockerfile`](./Dockerfile), which is configured to run
[`entrypoint.sh`](./entrypoint.sh) on startup. This script does several things:
including downloading the OPA configuration from DAS based on the environment
variables, setting up the environment for the sample apps and their
dependencies, and launching the sample app, OPA and Redoc (to serve the API
documentation). It then launches [`splitwatcher.sh`](./splitwatcher.sh), which
scripts [tmux](https://github.com/tmux/tmux/wiki) to display panes
[`tail`](https://linux.die.net/man/1/tail)-ing the OPA and sample app logs, and
also providing the user with an interactive terminal. In essence, this is a
bunch of window dressing on top of running an OPA server and then launching the
selected sample app.

Though it adds additional complexity to the end-to-end process, this approach
means that users can get up and running with an environment that will allow
them to experiment with the DAS quickly, and without having to install any
dependencies (other than Docker and git) on their system. This is ideal,
because users wishing to develop policies using DAS may not have the technical
expertise to set up a development environment suitable to run each sample app.
Even those users with the requisite knowledge may also prefer to get up and
running with the DAS more quickly so as to focus on the task at hand.
