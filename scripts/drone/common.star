load('scripts/drone/utils.star', 'ci_image')
load('scripts/drone/vault.star', 'from_secret')

def install_deps_step():
    return {
        'name': 'yarn-install',
        'image': ci_image,
        'commands': [
            '. ~/.init-nvm.sh',
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
            '. ~/.init-nvm.sh',
            'yarn build',
        ],
        'depends_on': [
            'yarn-install',
        ],
    }

def package_step(arch, name='', skip_chromium=False, override_output='', skip_errors=True):
    pkg_cmd = './scripts/package_target.sh {}'.format(arch)
    bpm_cmd = 'bin/grabpl build-plugin-manifest ./dist/'
    arc_cmd = './scripts/archive_target.sh {}'.format(arch)

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
            '. ~/.init-nvm.sh',
            pkg_cmd,
            bpm_cmd,
            arc_cmd,
        ],
        'depends_on': ['yarn-build'],
        'environment': {
            'GRAFANA_API_KEY': from_secret('grafana_api_key'),
        }
    }

    return step

def security_scan_step():
    return {
        'name': 'security-scan',
        'image': ci_image,
        'commands': [
            '. ~/.init-nvm.sh',
            'echo "Starting veracode scan..."',
            '# Increase heap size or the scanner will die.',
            'export _JAVA_OPTIONS=-Xmx4g',
            'mkdir -p ci/jobs/security_scan',
            'curl -sSL https://download.sourceclear.com/ci.sh | sh -s scan --skip-compile --quick --allow-dirty',
        ],
        'depends_on': ['yarn-build'],
        'environment': {
            'SRCCLR_API_TOKEN': from_secret('srcclr_api_token'),
        },
        'failure': 'ignore',
    }
