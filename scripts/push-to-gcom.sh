#!/bin/zsh

JSON=$(cat ./scripts/tmp/plugin.json)

echo $JSON
echo "Pushing..."

gcom /plugins -X POST -H "Content-Type: application/json" -d "$JSON"
