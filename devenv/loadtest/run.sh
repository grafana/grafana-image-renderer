#/bin/bash

cd "$(dirname $0)"

run() {
  local duration='15m'
  local url='http://localhost:3000'
  local vus='2'
  local iterationsOption=''

  while getopts ":d:i:u:v:" o; do
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
    esac
	done
	shift $((OPTIND-1))

  docker run \
    -it \
    --network=host \
    --mount type=bind,source=$PWD,destination=/src \
    -e URL=$url \
    --rm \
    loadimpact/k6:master run \
    --vus $vus \
    --duration $duration \
    $iterationsOption \
    //src/render_test.js
}

run "$@"
