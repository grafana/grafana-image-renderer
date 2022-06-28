export type SanitizerConfig = {
  verboseLogging: boolean;
};

export interface PluginConfig {
  plugin: {
    grpc: {
      host: string;
      port: number;
    };
  };
  rendering: SanitizerConfig;
}

const defaultSanitizerConfigConfig: SanitizerConfig = {
  verboseLogging: false,
};

export const defaultPluginConfig: PluginConfig = {
  plugin: {
    grpc: {
      host: '127.0.0.1',
      port: 0,
    },
  },
  rendering: defaultSanitizerConfigConfig,
};
