load('scripts/drone/utils.star', 'docker_image', 'ci_image')
load('scripts/drone/vault.star', 'from_secret')

def publish_gh_release():
    return {
        'name': 'publish_to_github',
        'image': 'cibuilds/github:0.13.0',
        'commands': [
            # Ensure necessary tools are installed
            'apk add --no-cache openssl curl jq',

            # Write the private key to a file
            'echo "$GITHUB_APP_PRIVATE_KEY" > private-key.pem',
            'chmod 600 private-key.pem',

            # Generate the JWT
            'NOW=$(date +%s)',
            'EXPIRATION=$(($NOW + 600))',
            'HEADER=$(printf \'{"alg":"RS256","typ":"JWT"}\' | openssl base64 -A | tr \'+/\' \'-_\' | tr -d \'=\')',
            'PAYLOAD=$(printf \'{"iat":%d,"exp":%d,"iss":"%s"}\' $NOW $EXPIRATION $GITHUB_APP_ID | openssl base64 -A | tr \'+/\' \'-_\' | tr -d \'=\')',
            'HEADER_PAYLOAD="$HEADER.$PAYLOAD"',
            'SIGNATURE=$(echo -n "$HEADER_PAYLOAD" | openssl dgst -sha256 -sign private-key.pem | openssl base64 -A | tr \'+/\' \'-_\' | tr -d \'=\')',
            'JWT="$HEADER_PAYLOAD.$SIGNATURE"',

            # Request the installation access token
            'RESPONSE=$(curl -s -X POST \\',
            '  -H "Authorization: Bearer $JWT" \\',
            '  -H "Accept: application/vnd.github+json" \\',
            '  https://api.github.com/app/installations/$GITHUB_INSTALLATION_ID/access_tokens)',

            # Extract the token from the response
            'GITHUB_TOKEN=$(echo $RESPONSE | jq -r \'.token\')',

            # Export the token for use in subsequent commands
            'export GITHUB_TOKEN',

            # Run your scripts
            './scripts/generate_md5sum.sh',
            './scripts/publish_github_release.sh',
        ],
        'environment': {
            # These are passed as secrets for security
            'GITHUB_APP_ID': from_secret('github_app_id'),
            'GITHUB_APP_PRIVATE_KEY': from_secret('github_app_private_key'),
            'GITHUB_INSTALLATION_ID': from_secret('github_app_installation_id'),
        },
        'depends_on': [
            'package-linux-x64-glibc',
            'package-darwin-x64-unknown',
            'package-win32-x64-unknown',
            'package-linux-x64-glibc-no-chromium',
            'package-alpine-x64-no-chromium',
        ],
    }

def publish_to_docker_master():
    step = publish_to_docker()
    step['name'] += '_master'
    step['commands'][0] += ' master'
    return step

def publish_to_docker_release():
    step = publish_to_docker()
    step['depends_on'] = ['publish_to_github']
    return step

def publish_to_docker():
    return {
        'name': 'publish_to_docker',
        'image': 'google/cloud-sdk:449.0.0',
        'environment': {
            'IMAGE_NAME': docker_image,
            'DOCKER_USER': from_secret('docker_user'),
            'DOCKER_PASS': from_secret('docker_pass'),
        },
        'commands': ['./scripts/build_push_docker.sh'],
        'volumes': [{'name': 'docker', 'path': '/var/run/docker.sock'}],
        'depends_on': ['yarn-test'],
    }

def publish_to_gcom():
    return {
        'name': 'publish_to_gcom',
        'image': ci_image,
        'commands': [
            '. ~/.init-nvm.sh',
            'yarn run create-gcom-plugin-json ${DRONE_COMMIT}',
            'yarn run push-to-gcom',
        ],
        'environment': {
            'GCOM_URL': from_secret('gcom_url'),
            'GCOM_UAGENT': from_secret('gcom_uagent'),
            'GCOM_PUBLISH_TOKEN': from_secret('gcom_publish_token'),
        },
        'depends_on': ['publish_to_github'],
    }
