#!/bin/sh

set -e
set -u

# We require the following environment variables:
#
# DAS_TOKEN - the token to be use when retrieving the OPA configuration
# DAS_SYSTEM - DAS system ID we are to pull from
# DAS_URL - the full URL of the DAS tenant we are to pull the system down from
# SAMPLE_APP - the name of the sample app to run
#
# The following additional environment variables are significant:
#
# API_PORT - port on which the API server should run (default: 8123)
# DOCS_PORT - port on which the API documentation server should run (default: 8080)
#
# valid sample apps are:
#
# * "go-httpsample"

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
set -e
if [ -z "$DOCS_PORT" ] ; then DOCS_PORT=8080; fi
set -e

# Update the OAPIv3.1 spec to report the correct port.
sed -i 's/http:\/\/localhost:8123/http:\/\/localhost:'"$API_PORT"'/g' /src/entitlements-samples/carinfostore.yml 

cd /src/entitlements-samples

# Serve the CarInfoStore API docs.
printf "launching documentation server... "
redoc-cli serve --port $DOCS_PORT ./carinfostore.yml > /var/log/redoc.log 2>&1 &
sleep 1
if ! ps aux | grep -v grep | grep -q redoc-cli ; then
	echo "FAIL"
	echo "redoc-cli is not running. Printing redoc-cli logs and exiting..."
	echo "--------"
	cat /var/log/redoc.log
	exit 1
fi


# Insert port information into the welcome message.
sed -i 's/API_PORT/'"$API_PORT"'/g' welcome.txt
sed -i 's/DOCS_PORT/'"$DOCS_PORT"'/g' welcome.txt

if [ "$SAMPLE_APP" = "go-httpsample" ] ; then
	cd go-httpsample

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

	printf "launching OPA server... "
	opa run --server --config-file=opa-conf.yaml >> /var/log/opa-server.log 2>&1 &
	echo "DONE"

	printf "launching carinfoserver... "
	./carinfoserver --port $API_PORT --opa http://localhost:8181/v1/data/main/main >> /var/log/carinfoserver.log 2>&1 &
	echo "DONE"

	echo "launching interactive monitor... "

	export FORCE_PS1="sample$ "
	export STARTDIR="/src/entitlements-samples/go-httpsample"
	export INJECT_COMMANDS="alias curl='curl -w \"\\n\"'"
	export WELCOME="/src/entitlements-samples/welcome.txt"
	sh /src/splitwatcher.sh /var/log/opa-server.log /var/log/carinfoserver.log

elif [ "$SAMPLE_APP" = "python-httpsample" ] ; then
	cd python-httpsample

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

	printf "launching OPA server... "
	opa run --server --config-file=opa-conf.yaml >> /var/log/opa-server.log 2>&1 &
	echo "DONE"

	printf "launching carinfoserver... "
	python3 server.py --port $API_PORT --opa http://localhost:8181/v1/data/main/main >> /var/log/carinfoserver.log 2>&1 &
	echo "DONE"

	echo "launching interactive monitor... "

	export FORCE_PS1="sample$ "
	export STARTDIR="/src/entitlements-samples/python-httpsample"
	export INJECT_COMMANDS="alias curl='curl -w \"\\n\"'"
	export WELCOME="/src/entitlements-samples/welcome.txt"
	sh /src/splitwatcher.sh /var/log/opa-server.log /var/log/carinfoserver.log

else
	echo "don't know how to run sample app '$SAMPLE_APP'" 1>&2
	exit 1
fi

