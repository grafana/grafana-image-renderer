load(
    'scripts/drone/utils.star',
    'pipeline',
    'ci_image',
)

def security_scan_step():
    return {
        'name': 'security-scan',
        'image': ci_image,
        'commands': [
            'echo "Starting veracode scan..."',
            'apk add curl',
            '# Increase heap size or the scanner will die.',
            'export _JAVA_OPTIONS=-Xmx4g',
            'mkdir -p ci/jobs/security_scan',
            'curl -sSL https://download.sourceclear.com/ci.sh | sh -s scan --skip-compile --quick --allow-dirty',
        ],
        'failure': 'ignore',
        'depends_on': [
        ],
    }


def security_pipeline():
    trigger = {
        'event': [
            'push',
        ],
        'branch': 'master',
    }

    return [
        pipeline(
            name='perform-srcclr-scan',
            trigger=trigger,
            steps=[
                security_scan_step(),
            ]
        ),
    ]
