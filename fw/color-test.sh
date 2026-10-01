#!/usr/bin/env bash

export REGISTRY="${REGISTRY:-http://localhost:8080}"

while true; do
	# A countdown may change before Registry observes the requested exact value.
	reg set ssr.watchdog 60
	reg set ssr.channel.1 0
	reg set ssr.channel.2 20
	reg set ssr.channel.3 40
	reg set ssr.channel.4 60
	reg set ssr.channel.5 80
	reg set ssr.channel.6 100
	sleep 1
done