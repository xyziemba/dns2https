#!/usr/bin/env bash

export PORT=8053

./dnsproxy -port $PORT >&2 &
DNSPID=$!
bats app-tests.bats
CODE=$?

kill -15 $DNSPID

exit $?