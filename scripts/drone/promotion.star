load('scripts/drone/utils.star', 'docker_image', 'ci_image')
load('scripts/drone/vault.star', 'from_secret')

def publish_gh_release():
    return {
        'name': 'publish_to_github',
        'image': 'cibuilds/github:0.13.0',
        'commands': [
            'sh scripts/generate_md5sum.sh',
            'sh scripts/publish_github_release.sh',
        ],
        'environment': {
            'GITHUB_TOKEN': from_secret('github_token'),
        },
        'depends_on': [
            'package-linux-x64-glibc',
            'package-darwin-x64-unknown',
            'package-win32-x64-unknown',
            'package-linux-x64-glibc-no-chromium',
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
        'image': 'google/cloud-sdk:412.0.0',
        'environment': {
            'IMAGE_NAME': docker_image,
            'DOCKER_USER': from_secret('docker_user'),
            'DOCKER_PASS': from_secret('docker_pass'),
        },
        'commands': ['sh scripts/build_push_docker.sh'],
        'volumes': [{'name': 'docker', 'path': '/var/run/docker.sock'}],
    }

def publish_to_gcom():
    return {
        'name': 'publish_to_gcom',
        'image': ci_image,
        'commands': [
            '. ~/.init-nvm.sh',
            'yarn run create-gcom-plugin-json ${DRONE_COMMIT}',
            'sh scripts/push-to-gcom.sh',
        ],
        'environment': {
            'GCOM_URL': from_secret('gcom_url'),
            'GCOM_UAGENT': from_secret('gcom_uagent'),
            'GCOM_PUBLISH_TOKEN': from_secret('gcom_publish_token'),
        },
        'depends_on': ['publish_to_github'],
    }
