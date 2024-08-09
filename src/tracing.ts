import {NodeSDK} from '@opentelemetry/sdk-node';
import {getNodeAutoInstrumentations} from '@opentelemetry/auto-instrumentations-node';
import {OTLPTraceExporter} from '@opentelemetry/exporter-trace-otlp-http';
import {SEMRESATTRS_SERVICE_NAME} from '@opentelemetry/semantic-conventions';
import {Resource} from '@opentelemetry/resources';
import {ConsoleLogger, Logger} from "./logger";
import {getServiceConfig} from "./utils";

// For troubleshooting, set the log level to DiagLogLevel.DEBUG
// const {diag, DiagConsoleLogger, DiagLogLevel} = require('@opentelemetry/api');
// diag.setLogger(new DiagConsoleLogger(), DiagLogLevel.DEBUG);

const config = getServiceConfig();
const log = new ConsoleLogger(config.service.logging);

if (config.service.tracing.enabled) {
    log.info('Starting tracing', 'traceExporterURL', config.service.tracing.exporterURL);
    const traceExporter = new OTLPTraceExporter({
        url:  config.service.tracing.exporterURL, // Change to your Jaeger or OTLP endpoint
    });

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
            .then(() => log.debug('Tracing terminated'))
            .catch((error) => log.error('Error terminating tracing', error))
            .finally(() => process.exit(0));
    });
}
