#!/bin/bash

set -e

# Ensure necessary tools are installed
apk add --no-cache openssl curl jq

# Write the private key to a file
echo "$GITHUB_APP_PRIVATE_KEY" > private-key.pem
chmod 600 private-key.pem

# Generate the JWT
NOW=$(date +%s)
EXPIRATION=$(($NOW + 600))
HEADER=$(printf '{"alg":"RS256","typ":"JWT"}' | openssl base64 -A | tr '+/' '-_' | tr -d '=')
PAYLOAD=$(printf '{"iat":%d,"exp":%d,"iss":"%s"}' $NOW $EXPIRATION $GITHUB_APP_ID | openssl base64 -A | tr '+/' '-_' | tr -d '=')
HEADER_PAYLOAD="$HEADER.$PAYLOAD"
SIGNATURE=$(echo -n "$HEADER_PAYLOAD" | openssl dgst -sha256 -sign ./private-key.pem | openssl base64 -A | tr '+/' '-_' | tr -d '=')
JWT="$HEADER_PAYLOAD.$SIGNATURE"

# Request the installation access token
RESPONSE=$(curl -s -X POST \
  -H "Authorization: Bearer $JWT" \
  -H "Accept: application/vnd.github+json" \
  https://api.github.com/app/installations/$GITHUB_INSTALLATION_ID/access_tokens)

# Extract the token from the response
GITHUB_TOKEN=$(echo $RESPONSE | jq -r '.token')

# Export the token for use in subsequent commands
export GITHUB_TOKEN
