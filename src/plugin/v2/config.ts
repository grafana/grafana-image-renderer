import { SecurityConfig } from '../../config/security';
import { defaultRenderingConfig, populateRenderingConfigFromEnv, Mode, RenderingConfig } from '../../config/rendering';

export interface PluginConfig {
  plugin: {
    grpc: {
      host: string;
      port: number;
    };
    security: SecurityConfig;
  };
  rendering: RenderingConfig;
}

export const defaultPluginConfig: PluginConfig = {
  plugin: {
    grpc: {
      host: '127.0.0.1',
      port: 0,
    },
    security: {
      authToken: '-',
    },
  },
  rendering: defaultRenderingConfig,
};

export function populatePluginConfigFromEnv(config: PluginConfig, env: NodeJS.ProcessEnv) {
  if (env['GF_PLUGIN_GRPC_HOST']) {
    config.plugin.grpc.host = env['GF_PLUGIN_GRPC_HOST'] as string;
  }

  if (env['GF_PLUGIN_GRPC_PORT']) {
    config.plugin.grpc.port = parseInt(env['GF_PLUGIN_GRPC_PORT'] as string, 10);
  }

  if (env['GF_PLUGIN_AUTH_TOKEN']) {
    const authToken = env['GF_PLUGIN_AUTH_TOKEN'] as string;
    config.plugin.security.authToken = authToken.includes(' ') ? authToken.split(' ') : authToken;
  }

  populateRenderingConfigFromEnv(config.rendering, env, Mode.Plugin);
}
