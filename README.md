# Entitlements Samples

This repository contains examples of using the Entitlements feature of Styra
DAS in various languages and settings.

## Exercises

TODO: put in some helpful exercises for the user to runt hrough.

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
