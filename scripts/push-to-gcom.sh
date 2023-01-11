#!/bin/zsh

JSON=$(cat ./scripts/tmp/plugin.json)

echo $JSON
echo "Pushing..."

curl -s -H "User-Agent: $GCOM_UAGENT" -H "Authorization: Bearer $GCOM_PUBLISH_TOKEN" "$GCOM_URL/plugins" -X POST -H "Content-Type: application/json" -d "$JSON"
