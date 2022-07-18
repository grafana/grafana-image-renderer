#!/bin/zsh

JSON=$(cat ./scripts/tmp/plugin.json)
gcom /plugins -X POST -H "Content-Type: application/json" -d "$JSON"
