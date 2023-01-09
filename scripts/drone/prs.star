load(
    'scripts/drone/utils.star',
    'pipeline',
)

load(
    'scripts/drone/grabpl.star',
    'download_grabpl_step',
)

load(
    'scripts/drone/step.star',
    'install_deps_step', 'build_step', 'package_step',
)

def prs_pipeline():
    trigger = {
        'event': [
            'pull_request',
        ],
    }

    return [
        pipeline(
            name='test-pr',
            trigger=trigger,
            steps=[
                download_grabpl_step(),
                install_deps_step(),
                build_step(),
                package_step(arch='linux-x64-glibc'),
                package_step(arch='darwin-x64-unknown'),
                package_step(arch='win32-x64-unknown'),
                package_step(arch='linux-x64-glibc', name='package-linux-x64-glibc-no-chromium', skip_chromium=True, override_output='plugin-linux-x64-glibc-no-chromium'),
            ]
        ),
    ]
