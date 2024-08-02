import { NodeSDK } from '@opentelemetry/sdk-node';
import { getNodeAutoInstrumentations } from '@opentelemetry/auto-instrumentations-node';
import { OTLPTraceExporter } from '@opentelemetry/exporter-trace-otlp-http';
import { SEMRESATTRS_SERVICE_NAME } from '@opentelemetry/semantic-conventions';
import { Resource } from '@opentelemetry/resources';
import { ConsoleLogger, PluginLogger } from './logger';
import {defaultServiceConfig, ServiceConfig} from "./service/config";

const traceExporter = new OTLPTraceExporter({
    url: 'http://localhost:14268/api/traces', // Change to your Jaeger or OTLP endpoint
});

const sdk = new NodeSDK({
    resource: new Resource({
        [SEMRESATTRS_SERVICE_NAME]: 'image-renderer-service', // Set your service name here
    }),
    traceExporter,
    instrumentations: [getNodeAutoInstrumentations()],
});

sdk.start();
let config: ServiceConfig = defaultServiceConfig;

const logger = new ConsoleLogger(config.service.logging);
logger.info('Starting tracing');
