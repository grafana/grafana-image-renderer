import {NodeSDK} from '@opentelemetry/sdk-node';
import {getNodeAutoInstrumentations} from '@opentelemetry/auto-instrumentations-node';
import {OTLPTraceExporter} from '@opentelemetry/exporter-trace-otlp-http';
import {SEMRESATTRS_SERVICE_NAME} from '@opentelemetry/semantic-conventions';
import {Resource} from '@opentelemetry/resources';
import {Logger} from "./logger";

// For troubleshooting, set the log level to DiagLogLevel.DEBUG
// const { diag, DiagConsoleLogger, DiagLogLevel } = require('@opentelemetry/api');
// diag.setLogger(new DiagConsoleLogger(), DiagLogLevel.DEBUG);

const traceExporterURL = 'http://localhost:4318/v1/traces'

const traceExporter = new OTLPTraceExporter({
    url: traceExporterURL, // Change to your Jaeger or OTLP endpoint
});

const sdk = new NodeSDK({
    resource: new Resource({
        [SEMRESATTRS_SERVICE_NAME]: 'image-renderer-service',
    }),
    traceExporter,
    instrumentations: [
        getNodeAutoInstrumentations({
        // only instrument fs if it is part of another trace
        '@opentelemetry/instrumentation-fs': {
          requireParentSpan: true,
        },
      })
    ],
});

export  function startTracing(log: Logger) {
    sdk.start();
    log.info('Starting tracing');

    process.on('SIGTERM', () => {
        sdk.shutdown()
            .then(() => log.debug('Tracing terminated'))
            .catch((error) => log.error('Error terminating tracing', error))
            .finally(() => process.exit(0));
    });
}
