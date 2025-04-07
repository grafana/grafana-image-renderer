load('scripts/drone/vault.star', 'gcr_pull_secret', 'gar_pull_secret')

ci_image = 'grafana/grafana-plugin-ci:1.9.0'
docker_image = 'grafana/grafana-image-renderer'
publisher_image = 'grafana/integration-grafana-publisher:v9'

def pipeline(
    name,
    trigger,
    steps,
    services=[],
    platform='linux',
    depends_on=[],
    environment=None,
    volumes=[],
):
    if platform != 'windows':
        platform_conf = {
            'platform': {'os': 'linux', 'arch': 'amd64'},
            # A shared cache is used on the host
            # To avoid issues with parallel builds, we run this repo on single build agents
            'node': {'type': 'no-parallel'},
        }
    else:
        platform_conf = {
            'platform': {
                'os': 'windows',
                'arch': 'amd64',
                'version': '1809',
            }
        }

    pipeline = {
        'kind': 'pipeline',
        'type': 'docker',
        'name': name,
        'trigger': trigger,
        'services': services,
        'steps': steps,
        'clone': {
            'retries': 3,
        },
        'volumes': [
            {
                'name': 'docker',
                'host': {
                    'path': '/var/run/docker.sock',
                },
            }
        ],
        'depends_on': depends_on,
        'image_pull_secrets': [gcr_pull_secret, gar_pull_secret],
    }
    if environment:
        pipeline.update(
            {
                'environment': environment,
            }
        )

    pipeline['volumes'].extend(volumes)
    pipeline.update(platform_conf)

    return pipeline
