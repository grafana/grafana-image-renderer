# Install / update dependencies
yarn install --frozen-lockfile

# Start Grafana
docker compose -f ./devenv/docker/test/docker-compose.yaml up -d

# Start testing
yarn jest
