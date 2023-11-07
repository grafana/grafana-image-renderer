import { ConsoleLogger } from '../logger';
import { Browser } from './browser';
import * as promClient from 'prom-client';

jest.mock('../logger')

const renderingConfig = {
  args: ['--no-sandbox', '--disable-gpu'],
  ignoresHttpsErrors: false,
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
  emulateNetworkConditions: false,
}

const browser = new Browser(renderingConfig, new ConsoleLogger({ level: 'info' }), {
  durationHistogram: new promClient.Histogram({
    name: 'grafana_image_renderer_step_duration_seconds',
    help: 'duration histogram of browser steps for rendering an image labeled with: step',
    labelNames: ['step'],
    buckets: [0.1, 0.3, 0.5, 1, 3, 5, 10, 20, 30],
  }),
});


test('validateRenderOptions should fail when passing an URL in socket mode', () => {
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
})

