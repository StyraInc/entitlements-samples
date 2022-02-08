# Entitlements Samples

This repository contains examples of using the Entitlements feature of Styra
DAS in various languages and settings.

## How to Run

This repository contains several different sample applications (listed below)
which demonstrate various methods by which to integrate OPA with your
application in various languages. For convenience, all of these sample
applications are bundled into a single docker image, and which application is
launched depends on the setting of the `SAMPLE_APP` environment variable (see
*Influential Environment Variables*).

Additionally, various environment variables need to be set which can be
retrieved from a DAS instance (tip: you can create your own DAS instance for
free via [this link](https://www.styra.com/das-free?hsLang=en)).

To get started, you need to first download the Docker image, or build it
locally.

To build locally:

```
$ docker build -t entitlements-samples .
```

To download the image:

```
$ docker pull ...
```

**TODO**: add instructions for downloading the image once we have a published
image.

An example of running the Python sample application would be:

```
$ docker run -it \
	-p 8080:8080 -p 8123:8123 -e DOCS_PORT=8080 -e API_PORT=8123 \
	-e SAMPLE_APP=python-httpsample \
	-e DAS_TOKEN='CHANGEME' \
	-e DAS_URL='https://CHANGEME.styra.com/' \
	-e DAS_SYSTEM='CHANGEME' \
	entitlements-samples
```

You can determine your DAS system ID (`DAS_SYSTEM`) by looking in the top-left
corner of the screen while you have the system in question selected in the DAS,
it should be a long sequence of letters and numbers such as
`ca8cef0d13134065bd7481f56f05537c`.

You can create a token (`DAS_TOKEN`) via Workspace->Settings->API Tokens.

Alternatively, if you view the quickstart for the "Entitlements" system type,
your DAS instance will automatically generate you an appropriate `docker run`
command with the necessary environment variables pre-filled (feel free to
change the `SAMPLE_APP` variable to try out different samples).

The sample applications that you can choose from are:

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
