#!/bin/sh
set -e
expected='{"key":1,"value":"foobar"}'
response=$(curl --silent localhost:8080/object/1)
if [ "${expected}" == "${response}" ]; then
echo "$0: OK"
else
echo "$0: KO expected: $expected got: $response"
exit 1
fi;