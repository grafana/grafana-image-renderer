pull_secret = 'dockerconfigjson'

def from_secret(secret):
    return {'from_secret': secret}

def vault_secret(name, path, key):
    return {
        'kind': 'secret',
        'name': name,
        'get': {
            'path': path,
            'name': key,
        },
    }

def secrets():
    return [
        vault_secret(pull_secret, 'secret/data/common/gcr', '.dockerconfigjson'),
        vault_secret('github_token', 'infra/data/ci/drone-plugins', 'github_token'),
        vault_secret('gcom_publish_token', 'infra/data/ci/drone-plugins', 'gcom_publish_token'),
        vault_secret('grafana_api_key', 'infra/data/ci/drone-plugins', 'grafana_api_key'),
        vault_secret('srcclr_api_token', 'infra/data/ci/drone-plugins', 'srcclr_api_token'),
    ]
