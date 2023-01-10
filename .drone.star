# To generate the .drone.yml file:
# 1. Modify the *.star definitions
# 2. Login to drone and export the env variables (token and server) shown here: https://drone.grafana.net/account
# 3. Run `make drone`
# More information about this process here: https://github.com/grafana/deployment_tools/blob/master/docs/infrastructure/drone/signing.md

load('scripts/drone/prs.star', 'prs_pipeline', 'master_pipeline')
load('scripts/drone/security.star', 'security_pipeline')
load('scripts/drone/vault.star', 'secrets')


def main(ctx):
    return (
        prs_pipeline()
        + master_pipeline()
        + security_pipeline()
        + secrets()
    )