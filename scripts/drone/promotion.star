load('scripts/drone/utils.star', 'docker_image')
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
    return
