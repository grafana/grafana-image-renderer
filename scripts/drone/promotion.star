load('scripts/drone/utils.star', 'docker_image', 'ci_image')
load('scripts/drone/vault.star', 'from_secret')

def publish_to_docker():
    return {
        'name': 'publish_to_docker',
        'image': 'google/cloud-sdk:412.0.0',
        'environment': {
            'IMAGE_NAME': docker_image,
            'DOCKER_USER': from_secret('docker_user'),
            'DOCKER_PASS': from_secret('docker_pass'),
        },
        'commands': [
            'sh scripts/build_push_docker.sh master',
        ],
        'volumes': [{'name': 'docker', 'path': '/var/run/docker.sock'}],
    }

def publish_release():
    return {
        'name': 'publish_to_github',
        'image': ci_image,
        'commands': [
            'apk add --update --no-cache jq',
            'sh scripts/generate_md5sum.sh',
            'sh scripts/publish_github_release.sh',
        ],
        'depends_on': [
            'package-linux-x64-glibc',
            'package-darwin-x64-unknown',
            'package-win32-x64-unknown',
            'package-linux-x64-glibc-no-chromium',
        ],
    }

def publish_to_grafana():
    return {
        'name': 'publish_to_grafana',
        'image': ci_image,
        'commands': [
            'yarn run create-gcom-plugin-json ${DRONE_COMMIT}',
            'sh scripts/push-to-gcom.sh',
        ],
    }
