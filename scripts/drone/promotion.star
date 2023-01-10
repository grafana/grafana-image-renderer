load('scripts/drone/utils.star', 'docker_image')

def publish_to_docker():
    return {
        'name': 'publish_to_docker',
        'environment': {
            'IMAGE_NAME': docker_image,
            'DOCKER_USER': {
                'from_secret': 'docker_username'
            },
            'DOCKER_PASS': {
                'from_secret': 'docker_password'
            }
        },
        'commands': [
            '../build_push_docker.sh master',
        ],
    }

def publish_release():
    return
