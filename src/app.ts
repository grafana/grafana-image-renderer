import { RenderGRPCPluginV2 } from './plugin/v2/grpc_plugin';
import { PluginLogger } from './logger';
import { defaultPluginConfig, PluginConfig } from './config';
import { serve } from './node-plugin';

async function main() {
  const env = Object.assign({}, process.env);

  const logger = new PluginLogger();
  const config: PluginConfig = defaultPluginConfig;
  populatePluginConfigFromEnv(config, env);

  await serve({
    handshakeConfig: {
      protocolVersion: 2,
      magicCookieKey: 'grafana_plugin_type',
      magicCookieValue: 'datasource',
    },
    versionedPlugins: {
      2: {
        renderer: new RenderGRPCPluginV2(config, logger),
      },
    },
    logger: logger,
    grpcHost: config.plugin.grpc.host,
    grpcPort: config.plugin.grpc.port,
  });
}

main().catch((err) => {
  console.error(err);
  process.exit(1);
});

function populatePluginConfigFromEnv(config: PluginConfig, env: NodeJS.ProcessEnv) {
  // Plugin env variables that needs to be initiated early
  if (env['GF_SANITIZER_PLUGIN_GRPC_HOST']) {
    config.plugin.grpc.host = env['GF_PLUGIN_GRPC_HOST'] as string;
  }

  if (env['GF_SANITIZER_PLUGIN_GRPC_PORT']) {
    config.plugin.grpc.port = parseInt(env['GF_PLUGIN_GRPC_PORT'] as string, 10);
  }
}
