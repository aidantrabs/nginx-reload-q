#!/usr/bin/env bash
set -euo pipefail

SOCK="/var/run/nginx-reload.sock"
METRICS="http://127.0.0.1:9111/metrics"

echo "--- single reload"
echo "RELOAD" | sudo socat - UNIX-CONNECT:$SOCK
sleep 1

echo ""
echo "--- unknown command"
echo "HELLO" | sudo socat - UNIX-CONNECT:$SOCK

echo ""
echo "--- burst (10 concurrent)"
for i in {1..10}; do
  echo "RELOAD" | sudo socat - UNIX-CONNECT:$SOCK &
done
wait
sleep 2

echo ""
echo "--- metrics"
curl -s $METRICS | python3 -m json.tool

echo ""
echo "done"
