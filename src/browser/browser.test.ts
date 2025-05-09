import { ConsoleLogger } from '../logger';
import { RenderOptions } from '../types';
import { Browser } from './browser';
import * as promClient from 'prom-client';

jest.mock('../logger');

const renderingConfig = {
  args: ['--no-sandbox', '--disable-gpu'],
  ignoresHttpsErrors: false,
  acceptLanguage: 'fr-CA',
  width: 1000,
  height: 500,
  deviceScaleFactor: 1,
  maxWidth: 3000,
  maxHeight: 3000,
  maxDeviceScaleFactor: 4,
  pageZoomLevel: 1,
  mode: 'default',
  clustering: {
    monitor: false,
    mode: 'browser',
    maxConcurrency: 5,
    timeout: 30,
  },
  verboseLogging: false,
  dumpio: false,
  timingMetrics: false,
  tracing: {
    url: '',
    serviceName: '',
  },
  emulateNetworkConditions: false,
};

const browser = new Browser(renderingConfig, new ConsoleLogger({ level: 'info' }), {
  durationHistogram: new promClient.Histogram({
    name: 'grafana_image_renderer_step_duration_seconds',
    help: 'duration histogram of browser steps for rendering an image labeled with: step',
    labelNames: ['step'],
    buckets: [0.1, 0.3, 0.5, 1, 3, 5, 10, 20, 30],
  }),
});

describe('Test validateRenderOptions', () => {
  it('should fail when passing a socket URL', () => {
    const fn = () => {
      browser.validateRenderOptions({
        url: 'socket://localhost',
        filePath: '',
        timeout: 0,
        renderKey: '',
        domain: '',
      });
    };

    expect(fn).toThrow(Error);
  });

  it('should use accept-language header if it exists', () => {
    let options: RenderOptions = {
      url: 'http://localhost',
      filePath: '',
      timeout: 0,
      renderKey: '',
      headers: {
        'Accept-Language': 'en-US',
      },
      domain: '',
    };

    browser.validateRenderOptions(options);

    expect(options.headers?.['Accept-Language']).toEqual('en-US');
  });

  it('should use acceptLanguage configuration if no header is given', () => {
    let options: RenderOptions = {
      url: 'http://localhost',
      filePath: '',
      timeout: 0,
      renderKey: '',
      domain: '',
    };

    browser.validateRenderOptions(options);

    expect(options.headers?.['Accept-Language']).toEqual(renderingConfig.acceptLanguage);
  });

  it('should use timeout option if given', () => {
    let options: RenderOptions = {
      url: 'http://localhost',
      filePath: '',
      timeout: 5,
      renderKey: '',
      domain: '',
    };

    browser.validateRenderOptions(options);

    expect(options.timeout).toEqual(5);
  });

  it('should use default timeout if none is given', () => {
    let options: RenderOptions = {
      url: 'http://localhost',
      filePath: '',
      timeout: 0,
      renderKey: '',
      domain: '',
    };

    browser.validateRenderOptions(options);

    expect(options.timeout).toEqual(30);
  });
});
