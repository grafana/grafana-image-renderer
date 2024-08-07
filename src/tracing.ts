import {NodeSDK} from '@opentelemetry/sdk-node';
import {getNodeAutoInstrumentations} from '@opentelemetry/auto-instrumentations-node';
import {OTLPTraceExporter} from '@opentelemetry/exporter-trace-otlp-http';
import {SEMRESATTRS_SERVICE_NAME} from '@opentelemetry/semantic-conventions';
import {Resource} from '@opentelemetry/resources';
import {defaultServiceConfig} from "./service/config";
import {ConsoleLogger} from "./logger";

const traceExporter = new OTLPTraceExporter({
    url: 'http://localhost:4318/v1/traces', // Change to your Jaeger or OTLP endpoint
});

const logger = new ConsoleLogger(defaultServiceConfig.service.logging);
logger.debug('Starting tracing');

const sdk = new NodeSDK({
    resource: new Resource({
        [SEMRESATTRS_SERVICE_NAME]: 'image-renderer-service',
    }),
    traceExporter,
    instrumentations: [getNodeAutoInstrumentations()],
});

sdk.start();

process.on('SIGTERM', () => {
    sdk.shutdown()
        .then(() => logger.debug('Tracing terminated'))
        .catch((error) => logger.error('Error terminating tracing', error))
        .finally(() => process.exit(0));
});
