load('scripts/drone/utils.star', 'pipeline')
load('scripts/drone/grabpl.star', 'download_grabpl_step')
load('scripts/drone/common.star', 'install_deps_step', 'build_step', 'package_step')
load('scripts/drone/promotion.star', 'publish_to_docker', 'publish_release', 'publish_to_grafana')

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
    return [
        pipeline(
            name='test-pr',
            trigger={
                'event': ['pull_request'],
            },
            steps=common_steps(True)
        ),
    ]

def master_pipeline():
    return [
        pipeline(
            name='test-master',
            trigger={
                'branch': ['master'],
                'event': ['push'],
            },
            steps=common_steps(False)
        )
    ]

def promotion_pipeline():
    trigger = {
        'target': ['release'],
        'event': ['promote'],
    }

    steps = common_steps(False) + [
        publish_release(),
        publish_to_docker(),
        publish_to_grafana(),
    ]

    return [
        pipeline(
            name='release',
            trigger=trigger,
            steps=steps
        )
    ]
