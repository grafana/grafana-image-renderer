load('scripts/drone/utils.star', 'pipeline')
load('scripts/drone/grabpl.star', 'download_grabpl_step')
load('scripts/drone/step.star', 'install_deps_step', 'build_step', 'package_step', 'install_release_deps_step', 'generate_md5_checksums')
load('scripts/drone/promotion.star', 'publish_to_docker', 'publish_release')

def common_steps(skip_errors):
    return [
        download_grabpl_step(),
        install_deps_step(),
        build_step(),
        package_step(arch='linux-x64-glibc', skip_errors=skip_errors),
        package_step(arch='darwin-x64-unknown', skip_errors=skip_errors),
        package_step(arch='win32-x64-unknown', skip_errors=skip_errors),
        package_step(arch='linux-x64-glibc', name='package-linux-x64-glibc-no-chromium', skip_chromium=True, override_output='plugin-linux-x64-glibc-no-chromium', skip_errors=skip_errors),
    ]

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
            steps=common_steps(True)
        ),
    ]

def master_pipeline():
    trigger = {
        'branch': [
            'master',
        ],
    }

    return [
        pipeline(
            name='test-master',
            trigger=trigger,
            steps=common_steps(False)
        )
    ]

def promotion_pipeline():
    trigger = {
        'target': ['release'],
        'event': ['promote'],
    }

    steps=[
        install_release_deps_step(),
        generate_md5_checksums(),
        publish_release(),
        publish_to_docker(),
    ]

    return [
        pipeline(
            name='release',
            trigger=trigger,
            steps=steps
        )
    ]
