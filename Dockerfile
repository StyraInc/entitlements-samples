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
# build-essential, git, and golang-go are all require to build OPA itself
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
	apt-get -qq --yes install build-essential curl git golang-go jq python3 python3-pip tmux vim-tiny nano tcpdump npm && \
	sh -c "ln -s '$(which vim.tiny)' /usr/local/bin/vim"

# We will build OPA from source rather than downloading the release binary.
# This is done in order to provide compatibility with M1 Macs. Although the
# AMD64 release binary will run on the M1 via Rosetta, it cannot run inside a
# docker container, which is a virtualized AARCH64 Linux instance.
RUN mkdir -p /src && \
	cd /src/ && \
	git clone https://github.com/open-policy-agent/opa.git && \
	cd opa && \
	git checkout v0.36.0 && \
	make install && \
	cp /root/go/bin/opa /usr/local/bin/opa

# Install an updated version of node. This is required to run redoc. Then
# install redoc itself.
RUN curl -sL https://deb.nodesource.com/setup_16.x | bash && \
    apt-get -qq --yes install nodejs && \
    npm i -g redoc-cli

# Copy in the source code for our samples, plus the entrypoint script
RUN mkdir -p /src/entitlements-samples/go-httpsample && \
	mkdir -p /src/entitlements-samples/go-embeddedsample & \
	mkdir -p /src/entitlements-samples/python-httpsample
COPY carinfostore.yml \
	welcome.txt \
	entrypoint.sh \
	splitwatcher.sh \
	data.json \
	/src/entitlements-samples
COPY python-httpsample/ /src/entitlements-samples/python-httpsample
COPY go-httpsample/ /src/entitlements-samples/go-httpsample
#COPY go-embeddedsample/ /src/entitlements-samples/go-embeddedsample

# Install the dependencies for the Python sample app, then compile the Go
# sample app (which will pull in it's deps automatically)
RUN pip3 install -r /src/entitlements-samples/python-httpsample/requirements.txt && \
	cd /src/entitlements-samples/go-httpsample && \
	cat go.mod && \
	go build -o carinfoserver ./cmd/carinfoserver

CMD ["sh", "/src/entitlements-samples/entrypoint.sh"]
