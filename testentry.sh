#!/bin/sh

set -e
set +u
if [ ! -z "$TEST" ] ; then
	set -u
	RETRIES=0
	MAX_RETRIES=48 # 240s
	while true ; do
		if [ "$RETRIES" -gt "$MAX_RETRIES" ] ; then
			sh -c "sleep 2; tmux send-keys \"echo 'exceeded maximum retries waiting for server to become available, tests are likely to fail!'\" Enter" &
			break
		fi

		sh -c "sleep 2; tmux send-keys \"echo 'waiting 5s for server to become available (retry $RETRIES/$MAX_RETRIES)'\" Enter" &
		sleep 5

		if curl -LSs "http://localhost:$API_PORT/cars" > /dev/null 2>&1 ; then
			sh -c "sleep 2; tmux send-keys \"echo 'server became ready after $RETRIES retries'\" Enter" &
			break
		fi

		RETRIES="$(expr $RETRIES + 1)"

	done
	sh -c "sleep 2; tmux send-keys \"pytest /src/entitlements-samples/tests\" Enter" &
fi
set -u
