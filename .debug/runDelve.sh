#!/bin/bash

# print env
echo "Runtime Environment:"
printf "\\tCFAPI_URL=%s\\n" "$CFAPI_URL"
printf "\\tSTORE_HOST=%s\\n" "$STORE_HOST"
printf "\\tSTORE_PORT=%s\\n" "$STORE_PORT"
printf "\\tSTORE_PASSWORD=%s\\n" "$STORE_PASSWORD"
printf "\\tTYPES_CONFIG=%s\\n" "$TYPES_CONFIG"

# run debugger
# dlv debug --listen=localhost:2345 --headless=true --log=true ./cmd -- --log-level=debug --skip-monitor=true server --port=8080
dlv debug --listen=localhost:2345 --headless=true --log=true ./cmd -- --log-level=debug --json=false --skip-monitor=true --config="$TELEPRESENCE_ROOT/$TYPES_CONFIG" server