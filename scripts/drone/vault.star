gcr_pull_secret = "gcr"
gar_pull_secret = "gar"

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
        vault_secret(gcr_pull_secret, 'secret/data/common/gcr', '.dockerconfigjson'),
        vault_secret('github_app_id', 'ci/data/repo/grafana/grafana-image-renderer/github-app', 'app-id'),
        vault_secret('github_app_private_key', 'ci/data/repo/grafana/grafana-image-renderer/github-app', 'private-key'),
        vault_secret('github_app_installation_id', 'ci/data/repo/grafana/grafana-image-renderer/github-app', 'app-installation-id'),
        vault_secret('gcom_publish_token', 'infra/data/ci/drone-plugins', 'gcom_publish_token'),
        vault_secret('grafana_api_key', 'infra/data/ci/drone-plugins', 'grafana_api_key'),
        vault_secret('srcclr_api_token', 'infra/data/ci/drone-plugins', 'srcclr_api_token'),
        vault_secret(gar_pull_secret, 'secret/data/common/gar', '.dockerconfigjson'),
        vault_secret('docker_username', 'ci/data/common/dockerhub', 'username'),
        vault_secret('docker_password', 'ci/data/common/dockerhub', 'password'),
    ]
