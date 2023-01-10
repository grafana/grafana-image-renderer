load('scripts/drone/utils.star', 'docker_image')
load('scripts/drone/vault.star', 'from_secret')

def publish_to_docker():
    return {
        'name': 'publish_to_docker',
        'image': 'docker:20.10.22-dind-alpine3.17',
        'environment': {
            'IMAGE_NAME': docker_image,
            'DOCKER_USER': from_secret('docker_user'),
            'DOCKER_PASS': from_secret('docker_pass'),
        },
        'commands': [
            'sh scripts/build_push_docker.sh master',
        ],
    }

def publish_release():
    return
