import * as grpc from '@grpc/grpc-js';
import { Logger } from '../../logger';

export interface GrpcPlugin {
  grpcServer(server: grpc.Server): Promise<void>;
}

// CoreProtocolVersion is the ProtocolVersion of the plugin system itself.
// We will increment this whenever we change any protocol behavior. This
// will invalidate any prior plugins but will at least allow us to iterate
// on the core in a safe way. We will do our best to do this very
// infrequently.
export const coreProtocolVersion = 1;

// HandshakeConfig is the configuration used by client and servers to
// handshake before starting a plugin connection. This is embedded by
// both ServeConfig and ClientConfig.
//
// In practice, the plugin host creates a HandshakeConfig that is exported
// and plugins then can easily consume it.
export interface HandshakeConfig {
  // ProtocolVersion is the version that clients must match on to
  // agree they can communicate. This should match the ProtocolVersion
  // set on ClientConfig when using a plugin.
  // This field is not required if VersionedPlugins are being used in the
  // Client or Server configurations.
  protocolVersion: number;
  // MagicCookieKey and value are used as a very basic verification
  // that a plugin is intended to be launched. This is not a security
  // measure, just a UX feature. If the magic cookie doesn't match,
  // we show human-friendly output.
  magicCookieKey: string;
  magicCookieValue: string;
}

export interface PluginSet {
  [key: string]: GrpcPlugin;
}

export interface VersionedPluginSet {
  [key: number]: PluginSet;
}

export interface ServeConfig {
  // HandshakeConfig is the configuration that must match clients.
  handshakeConfig: HandshakeConfig;

  // Plugins are the plugins that are served.
  // The implied version of this PluginSet is the Handshake.ProtocolVersion.
  plugins?: PluginSet;

  // VersionedPlugins is a map of PluginSets for specific protocol versions.
  // These can be used to negotiate a compatible version between client and
  // server. If this is set, Handshake.ProtocolVersion is not required.
  versionedPlugins?: VersionedPluginSet;

  // GRPCServer should be non-nil to enable serving the plugins over
  // gRPC. This is a function to create the server when needed with the
  // given server options. The server options populated by go-plugin will
  // be for TLS if set. You may modify the input slice.
  //
  // Note that the grpc.Server will automatically be registered with
  // the gRPC health checking service. This is not optional since go-plugin
  // relies on this to implement Ping().
  grpcServer?(): grpc.Server;

  // Logger is used to pass a logger into the server. If none is provided the
  // server will create a default logger.
  // Logger hclog.Logger

  grpcHost?: string;
  grpcPort?: number;

  logger?: Logger;
}
