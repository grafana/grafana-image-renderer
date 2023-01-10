load('scripts/drone/utils.star', 'ci_image')
load('scripts/drone/grabpl.star', 'download_grabpl_step')
load('scripts/drone/promotion.star', 'publish_to_docker')
load('scripts/drone/utils.star', 'pipeline')
load('scripts/drone/vault.star', 'from_secret')

def install_deps_step():
    return {
        'name': 'yarn-install',
        'image': ci_image,
        'commands': [
            'yarn install --frozen-lockfile --no-progress',
        ],
        'depends_on': [
            'grabpl',
        ],
    }

def build_step():
    return {
        'name': 'yarn-build',
        'image': ci_image,
        'commands': [
            'yarn build',
        ],
        'depends_on': [
            'yarn-install',
        ],
    }

def package_step(arch, name='', skip_chromium=False, override_output='', skip_errors=True):
    pkg_cmd = 'sh scripts/package_target.sh {}'.format(arch)
    bpm_cmd = 'bin/grabpl build-plugin-manifest ./dist/'
    arc_cmd = 'sh scripts/archive_target.sh {}'.format(arch)

    if skip_chromium:
        pkg_cmd += ' true {}'.format(override_output)
        bpm_cmd += '{}'.format(override_output)
        arc_cmd += ' {}'.format(override_output)
    else:
        bpm_cmd += 'plugin-{}'.format(arch)

    if skip_errors:
        bpm_cmd += ' || true'

    if name == '':
        name = 'package-{}'.format(arch)

    step = {
        'name': name,
        'image': ci_image,
        'commands': [
            pkg_cmd,
            bpm_cmd,
            arc_cmd,
        ],
        'depends_on': [
            'yarn-build',
        ],
        'environment': {
            'GRAFANA_API_KEY': from_secret('grafana_api_key'),
        }
    }

    return step
