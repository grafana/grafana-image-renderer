#/bin/bash

cd "$(dirname $0)"

run() {
  local duration='15m'
  local url='http://localhost:3000'
  local authToken=''
  local vus='2'
  local iterationsOption=''

  while getopts ":d:i:u:v:a:" o; do
    case "${o}" in
				d)
            duration=${OPTARG}
            ;;
        i)
            iterationsOption="--iterations ${OPTARG}"
            ;;
        u)
            url=${OPTARG}
            ;;
        v)
            vus=${OPTARG}
            ;;
        a)
            authToken=${OPTARG}
            ;;
    esac
	done
	shift $((OPTIND-1))

  docker run \
    -it \
    --network=host \
    --mount type=bind,source=$PWD,destination=/src \
    -e URL=$url \
    -e AUTH_TOKEN=$authToken \
    --rm \
    grafana/k6:master run \
    --vus $vus \
    --duration $duration \
    $iterationsOption \
    //src/render_test.js
}

run "$@"
