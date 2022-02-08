#!/bin/sh

# Copyright 2022 Styra Inc. All rights reserved.
# Use of this source code is governed by an Apache2
# license that can be found in the LICENSE file.

# For information on influential environment variables, see the README.

set -e
set -u

# We need to be running in an interactive terminal or else tmux will fail.
#
# Note that it dosen't seem possible to detect the case where the user runs
# Docker with -i only, since the TTY has in fact been allocated in that case.
if [ ! -t 0 ] ; then
	echo "standard input does not appear to be an interactive tty, please run docker with '-it'"
	exit 1
fi

if [ ! -t 1 ] ; then
	echo "standard output does not appear to be an interactive tty, please run docker with '-it'"
	exit 1
fi

# Validate required environment variables.
set +u
if [ -z "$DAS_TOKEN" ] ; then
	echo "environment variable DAS_TOKEN was not provided"
	exit 1
fi

if [ -z "$DAS_URL" ] ; then
	echo "environment variable DAS_URL was not provided"
	exit 1
fi

if [ -z "$DAS_SYSTEM" ] ; then
	echo "environment variable DAS_SYSTEM was not provided"
	exit 1
fi

if [ -z "$SAMPLE_APP" ] ; then
	echo "environment variable SAMPLE_APP was not provided"
	exit 1
fi

# Set default values for optional environment variables.
if [ -z "$API_PORT" ] ; then API_PORT=8123 ; fi
if [ -z "$DOCS_PORT" ] ; then DOCS_PORT=8080; fi
set -u

# Update the OAPIv3.1 spec to report the correct port.
sed -i 's/http:\/\/localhost:8123/http:\/\/localhost:'"$API_PORT"'/g' /src/entitlements-samples/carinfostore.yml

cd /src/entitlements-samples

# Serve the CarInfoStore API docs.
printf "launching documentation server... "
#redoc-cli serve --port $DOCS_PORT ./carinfostore.yml > /var/log/redoc.log 2>&1 &

# XXX: temporary hack because redoc-cli broke
redoc-cli bundle carinfostore.yml   # redirecting this to a log file breaks it for some reason??
mkdir /tmp/docs
mv redoc-static.html /tmp/docs/index.html
cd /tmp/docs/
python3 -m http.server 8080 >> /var/log/redoc.log 2>&1 &
cd /src/entitlements-samples

#sleep 1
#if ! ps aux | grep -v grep | grep -q redoc-cli ; then
#        echo "FAIL"
#        echo "redoc-cli is not running. Printing redoc-cli logs and exiting..."
#        echo "--------"
#        cat /var/log/redoc.log
#        exit 1
#fi
echo "DONE"


# Insert port information into the welcome message.
sed -i 's/API_PORT/'"$API_PORT"'/g' welcome.txt
sed -i 's/DOCS_PORT/'"$DOCS_PORT"'/g' welcome.txt

# OPA endpoint we should have the sample app use.
OPA_URL="http://localhost:8181/v1/data/main/main"

# Where should we 'cd' before running the sample app?
TARGET_DIR=/

# What command should we use to run the sample app?
RUN_COMMAND="echo 'you should enver see this'"

# If YES, launch OPA as a sidecar, otherwise we assume the application is using
# a built-in SDK.
LAUNCH_OPA=YES

# Detect the sample app and set our configuration appropriately.
if [ "$SAMPLE_APP" = "go-httpsample" ] ; then
	TARGET_DIR=/src/entitlements-samples/go-sample
	RUN_COMMAND="./carinfoserver --mode http --port $API_PORT --opa '$OPA_URL'"


elif [ "$SAMPLE_APP" = "python-httpsample" ] ; then
	TARGET_DIR=/src/entitlements-samples/python-httpsample
	RUN_COMMAND="python3 server.py --port $API_PORT --opa '$OPA_URL'"

elif [ "$SAMPLE_APP" = "go-sdksample" ] ; then
	TARGET_DIR=/src/entitlements-samples/go-sample
	RUN_COMMAND="./carinfoserver --mode sdk --port $API_PORT --config '$TARGET_DIR/opa-conf.yaml'"
	LAUNCH_OPA=NO

elif [ "$SAMPLE_APP" = "entz-playground" ] ; then
	TARGET_DIR=/src/entitlements-samples/go-sample
	RUN_COMMAND="./carinfoserver --mode sdk --playground --port $API_PORT --config '$TARGET_DIR/opa-conf.yaml'"
	LAUNCH_OPA=NO

else
	echo "don't know how to run sample app '$SAMPLE_APP'" 1>&2
	exit 1
fi

cd "$TARGET_DIR"

printf "downloading OPA configuration... "
# Note that we need to use curl -L, since DAS_URL may have a trailing
# /, in which case we need to pick up the HTTP 301.
curl -LSs -H "Authorization: Bearer $DAS_TOKEN" -o opa-conf.yaml "$DAS_URL/v1/systems/$DAS_SYSTEM/assets/opaconfig.yaml"
if [ "$(wc -l < opa-conf.yaml)" -lt 2 ] ; then
	echo "FAIL"
	echo "opa-conf.yaml is empty! Something is wrong."
	echo "running curl again with more verbose output, then exiting... "
	echo "-------------"
	set -x
	curl -Li -H "Authorization: Bearer $DAS_TOKEN" "$DAS_URL/v1/systems/$DAS_SYSTEM/assets/opaconfig.yaml"
	exit 1
fi
echo "DONE"

if [ "$LAUNCH_OPA" = "YES" ] ; then
	printf "launching OPA server... "
	opa run --server --config-file=opa-conf.yaml >> /var/log/opa-server.log 2>&1 &
	echo "DONE"
fi

printf "launching $SAMPLE_APP... "
set +u
if [ -z "$TEST" ] ; then
	# the test scripts wants a blank database
	cp /src/entitlements-samples/data.json ./
fi
set -u
sh -c "$RUN_COMMAND" >> /var/log/carinfoserver.log 2>&1 &
echo "DONE"

echo "launching interactive monitor... "

export FORCE_PS1="sample$ "
export STARTDIR="$TARGET_DIR"
export INJECT_COMMANDS="alias curl='curl -w \"\\n\"'"
export WELCOME="/src/entitlements-samples/welcome.txt"

set +u
if [ ! -z "$TEST" ] ; then
	sh -c "sleep 2 ; tmux send-keys \"pytest /src/entitlements-samples/tests\" Enter" &
fi
set -u

if [ "$LAUNCH_OPA" = "YES" ] ; then
	sh /src/entitlements-samples/splitwatcher.sh /var/log/opa-server.log /var/log/carinfoserver.log
else
	sh /src/entitlements-samples/splitwatcher.sh /var/log/carinfoserver.log
fi

exit 0
