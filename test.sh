#!/bin/bash

set -x

# host='localhost'
# url='http://localhost:8080'
host='youlistenhere.com'
url='http://youlistenhere.com'

randy() {
    echo "$RANDOM % 10" | bc
}

clienty() {
    curl -i -N -s -S \
    -H "Connection: Upgrade" \
    -H "Upgrade: websocket" \
    -H "Sec-WebSocket-Version: 13" \
    -H "Sec-WebSocket-Key: hihihi" \
    -H "Host: $host" \
    -H "Origin: $url" \
    $url/websocket \
    >/dev/null &
    echo $!
}

for _ in range{1..40}; do 
    {
      sleep "$(randy)"
      pid=$(clienty)
      sleep "$(randy)"
      kill "$pid" 2>/dev/null
    } &
done
