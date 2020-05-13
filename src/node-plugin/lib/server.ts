import * as grpc from '@grpc/grpc-js';
import * as protoLoader from '@grpc/proto-loader';
import { coreProtocolVersion, PluginSet, VersionedPluginSet, ServeConfig } from './types';
import { PluginLogger } from '../../logger';

export const healthPackageDef = protoLoader.loadSync(__dirname + '/../../../proto/health.proto', {
  keepCase: true,
  longs: String,
  enums: String,
  defaults: true,
  oneofs: true,
});

export const healthProtoDescriptor = grpc.loadPackageDefinition(healthPackageDef);

export const defaultGRPCServer = () => new grpc.Server();

interface ProtocolNegotiation {
  protoVersion: number;
  pluginSet: PluginSet;
}

const protocolVersion = (opts: ServeConfig): ProtocolNegotiation => {
  let protoVersion = opts.handshakeConfig.protocolVersion;
  const pluginSet = opts.plugins;
  const env = Object.assign({}, process.env);

  let clientVersions: number[] = [];
  if (env['PLUGIN_PROTOCOL_VERSIONS']) {
    const protocolVersions = (env['PLUGIN_PROTOCOL_VERSIONS'] as string).split(',');
    for (let n = 0; n < protocolVersions.length; n++) {
      const protocolVersion = parseInt(protocolVersions[n], 10);
      clientVersions.push(protocolVersion);
    }
  }

  // We want to iterate in reverse order, to ensure we match the newest
  // compatible plugin version.
  clientVersions = clientVersions.sort().reverse();

  // set the old un-versioned fields as if they were versioned plugins
  if (!opts.versionedPlugins) {
    opts.versionedPlugins = {} as VersionedPluginSet;
  }

  if (pluginSet) {
    opts.versionedPlugins[protoVersion] = pluginSet;
  }

  // Sort the versions to make sure we match the latest first
  let versions: number[] = [];
  for (let n = 0; n < Object.keys(opts.versionedPlugins).length; n++) {
    const version = Object.keys(opts.versionedPlugins)[n];
    versions.push(parseInt(version, 10));
  }

  versions = versions.sort().reverse();
  let versionedPluginSet: PluginSet = {};

  for (let n = 0; n < versions.length; n++) {
    const version = versions[n];
    // Record each version, since we guarantee that this returns valid
    // values even if they are not a protocol match.
    protoVersion = version;
    versionedPluginSet = opts.versionedPlugins[version];

    for (let i = 0; i < clientVersions.length; i++) {
      const clientVersion = clientVersions[i];
      if (clientVersion === protoVersion) {
        return {
          protoVersion,
          pluginSet: versionedPluginSet,
        };
      }
    }
  }

  return {
    protoVersion,
    pluginSet: versionedPluginSet,
  };
};

export const serve = async (opts: ServeConfig) => {
  const env = Object.assign({}, process.env);
  opts.logger = opts.logger || new PluginLogger();

  if (opts.handshakeConfig.magicCookieKey === '' || opts.handshakeConfig.magicCookieValue === '') {
    throw new Error(
      'Misconfigured ServeConfig given to serve this plugin: no magic cookie key or value was set. Please notify the plugin author and report this as a bug.'
    );
  }

  if (env[opts.handshakeConfig.magicCookieKey] !== opts.handshakeConfig.magicCookieValue) {
    throw new Error(
      'This binary is a plugin. These are not meant to be executed directly. Please execute the program that consumes these plugins, which will load any plugins automatically'
    );
  }

  // negotiate the version and plugins
  // start with default version in the handshake config
  const { protoVersion, pluginSet } = protocolVersion(opts);

  const server = new grpc.Server();
  const grpcHealthV1: any = healthProtoDescriptor['grpc']['health']['v1'];
  server.addService(grpcHealthV1.Health.service, {
    check: (_: any, callback: any) => {
      callback(null, { status: 'SERVING' });
    },
  });

  // Register all plugins onto the gRPC server.
  for (const key in pluginSet) {
    if (pluginSet.hasOwnProperty(key)) {
      const p = pluginSet[key];
      await p.grpcServer(server);
    }
  }

  opts.grpcHost = opts.grpcHost || '127.0.0.1';
  opts.grpcPort = opts.grpcPort || 0;

  return new Promise<void>((resolve, reject) => {
    const address = `${opts.grpcHost}:${opts.grpcPort}`;
    server.bindAsync(address, grpc.ServerCredentials.createInsecure(), (error: Error | null, port: number) => {
      if (error) {
        reject(error);
      }
      if (port === 0) {
        reject(new Error(`failed to bind address=${address}, boundPortNumber=${port}`));
      }

      server.start();
      console.log(`${coreProtocolVersion}|${protoVersion}|tcp|${opts.grpcHost}:${port}|grpc`);
      resolve();
    });
  });
};
