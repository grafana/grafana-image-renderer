grabpl_version = 'v3.0.20'
curl_image = 'byrnedo/alpine-curl:0.1.8'
wix_image = 'grafana/ci-wix:0.1.1'

def download_grabpl_step(platform="linux"):
    if platform == 'windows':
        return {
            'name': 'grabpl',
            'image': wix_image,
            'commands': [
                '$$ProgressPreference = "SilentlyContinue"',
                'Invoke-WebRequest https://grafana-downloads.storage.googleapis.com/grafana-build-pipeline/{}/windows/grabpl.exe -OutFile grabpl.exe'.format(
                    grabpl_version
                ),
            ],
        }

    return {
        'name': 'grabpl',
        'image': curl_image,
        'commands': [
            'mkdir -p bin',
            'curl -fL -o bin/grabpl https://grafana-downloads.storage.googleapis.com/grafana-build-pipeline/{}/grabpl'.format(
                grabpl_version
            ),
            'chmod +x bin/grabpl',
        ],
    }
