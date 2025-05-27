import { NodeSDK } from '@opentelemetry/sdk-node';
import { getNodeAutoInstrumentations } from '@opentelemetry/auto-instrumentations-node';
import { OTLPTraceExporter } from '@opentelemetry/exporter-trace-otlp-http';
import { ATTR_SERVICE_NAME } from '@opentelemetry/semantic-conventions';
import { Resource } from '@opentelemetry/resources';
import { diag, DiagConsoleLogger, DiagLogLevel } from '@opentelemetry/api';

import { Logger } from './logger';
import { getConfig } from './config/config';

const config = getConfig();
let sdk;
if (config.rendering.tracing.url) {
  sdk = initTracing(config.rendering.tracing.url, config.rendering.verboseLogging);
}

function initTracing(exporterURL: string, verboseLogging: boolean = false) {
  if (verboseLogging) {
    diag.setLogger(new DiagConsoleLogger(), DiagLogLevel.DEBUG);
  }

  const traceExporter = new OTLPTraceExporter({
    url: exporterURL,
  });

  return new NodeSDK({
    resource: new Resource({
      [ATTR_SERVICE_NAME]: config.rendering.tracing.serviceName || 'grafana-image-renderer',
    }),
    traceExporter,
    instrumentations: [
      getNodeAutoInstrumentations({
        // only instrument fs if it is part of another trace
        '@opentelemetry/instrumentation-fs': {
          requireParentSpan: true,
        },
      }),
    ],
  });
}

export function startTracing(log: Logger) {
  sdk.start();
  log.info('Starting tracing');

  process.on('SIGTERM', () => {
    sdk
      .shutdown()
      .then(() => log.debug('Tracing terminated'))
      .catch((error) => log.error('Error terminating tracing', 'err', error))
      .finally(() => process.exit(0));
  });
}
