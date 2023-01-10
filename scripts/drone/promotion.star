load('scripts/drone/utils.star', 'docker_image')
load('scripts/drone/vault.star', 'from_secret')

def publish_to_docker():
    return {
        'name': 'publish_to_docker',
        'image': 'google/cloud-sdk',
        'environment': {
            'IMAGE_NAME': docker_image,
            'DOCKER_USER': from_secret('docker_user'),
            'DOCKER_PASS': from_secret('docker_user'),
        },
        'commands': [
            'sh scripts/build_push_docker.sh master',
        ],
    }

def publish_release():
    return
