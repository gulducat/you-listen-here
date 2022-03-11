#!/bin/bash

# set -x

randy() {
    echo "$RANDOM % 10" | bc
}

clienty() {
    sleep "$(randy)"
    curl -Ss localhost:8080/stream >/dev/null &
    echo $!
}

for _ in range{1..150}; do 
    {
      pid=$(clienty)
      sleep "$(randy)"
      kill "$pid" 2>/dev/null
    } &
done
