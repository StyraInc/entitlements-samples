# Copyright 2022 Styra Inc. All rights reserved.
# Use of this source code is governed by an Apache2
# license that can be found in the LICENSE file.

# This Dockerfile defines the base environment for all of the
# entitlements-samples to run it. It embeds each of them and determines which
# one to call based on an environment variable (see `entrypoint.py`).

FROM ubuntu:20.04

# Prevent apt from prompting us. Sometimes it will try to ask for things like
# timezones or to accept licenses. Since the Dockerfile is built
# non-interactively, this can cause docker build to hang. Note that we also use
# "-qq --yes" as an argument to apt-get install for the same reason.
ARG DEBIAN_FRONTEND=noninteractive

# software-properties-common is required to get the add-apt-repository command
#
# ppa:longlseep/golang-backports gives us a convienient way to install and
# up-to-date version of the Go compiler, since the one Ubuntu ships is too old
# for us.
#
# python3 and python3-pip are used for the Python example.
#
# tmux is needed for splitwatcher.
#
# npm is needed for redoc-cli, which we need to build and serve the Swagger
# documentation
#
# The other programs installed for convienience if we need to shell in to the
# container for debugging.
RUN apt-get update && \
	apt-get -qq --yes install software-properties-common && \
	yes | add-apt-repository ppa:longsleep/golang-backports && \
	apt-get update && \
	apt-get -qq upgrade --yes && \
	apt-get -qq --yes install curl git golang-go jq python3 python3-pip tmux vim-tiny nano tcpdump npm && \
	sh -c "ln -s '$(which vim.tiny)' /usr/local/bin/vim"

# Install OPA from static binary according to the detected CPU arch.
RUN OPA_VERSION=v0.37.2 && \
    URL="ERROR" && \
    if   [ "$(arch)" = "aarch64" ] ; then URL="https://github.com/open-policy-agent/opa/releases/download/$OPA_VERSION/opa_linux_arm64_static" ; \
    elif [ "$(arch)" = "x86_64"  ] ; then URL="https://github.com/open-policy-agent/opa/releases/download/$OPA_VERSION/opa_linux_amd64_static"  ; \
    else echo "Don't know where to get OPA for architecture '$(arch)'" ; exit 1 ; fi && \
    curl -LSs -o /usr/local/bin/opa "$URL" && chmod +x /usr/local/bin/opa

# Install an updated version of node. This is required to run redoc. Then
# install redoc itself.
RUN curl -sL https://deb.nodesource.com/setup_16.x | bash && \
    apt-get -qq --yes install nodejs && \
    npm i -g redoc-cli

# Copy in the source code for our samples, plus the entrypoint script
RUN mkdir -p /src/entitlements-samples/go-sample && \
	mkdir -p /src/entitlements-samples/python-httpsample && \
	mkdir -p /src/entitlements-samples/tests
COPY carinfostore.yml \
	welcome.txt \
	entrypoint.sh \
	splitwatcher.sh \
	data.json \
	/src/entitlements-samples
COPY python-httpsample/ /src/entitlements-samples/python-httpsample
COPY go-sample/ /src/entitlements-samples/go-sample
COPY tests/ /src/entitlements-samples/tests

# Install the dependencies for the Python sample app, then compile the Go
# sample app (which will pull in it's deps automatically).
#
# pytest and pytest-order are needed to run the test suite.
RUN pip3 install -r /src/entitlements-samples/python-httpsample/requirements.txt && \
	cd /src/entitlements-samples/go-sample && \
	go mod tidy && \
	go build -o carinfoserver ./cmd/carinfoserver && \
	pip3 install pytest pytest-order

CMD ["sh", "/src/entitlements-samples/entrypoint.sh"]
