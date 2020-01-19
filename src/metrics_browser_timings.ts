import { Browser as PuppeteerBrowser, Page } from 'puppeteer';
import * as _ from 'lodash';
import * as promClient from 'prom-client';
import { Histogram } from 'prom-client';

export class MetricsBrowserTimings {
  durationHistogram: Histogram;

  constructor() {
    this.durationHistogram = new promClient.Histogram({
      name: 'grafana_image_renderer_step_duration_seconds',
      help: 'duration histogram of browser steps for rendering an image labeled with: step',
      labelNames: ['step'],
      buckets: [0.3, 0.5, 1, 2, 3, 5],
    });
  }

  async launch(callback: () => Promise<PuppeteerBrowser>) {
    const timer = this.durationHistogram.startTimer({ step: 'launch' });
    const browser = await callback();
    timer();
    return browser;
  }

  async newPage(callback: () => Promise<Page>) {
    const timer = this.durationHistogram.startTimer({ step: 'newPage' });
    const page = await callback();
    timer();
    return page;
  }

  async navigate(callback: () => Promise<void>) {
    const timer = this.durationHistogram.startTimer({ step: 'navigate' });
    await callback();
    timer();
  }

  async panelsRendered(callback: () => Promise<void>) {
    const timer = this.durationHistogram.startTimer({ step: 'panelsRendered' });
    await callback();
    timer();
  }

  async screenshot(callback: () => Promise<void>) {
    const timer = this.durationHistogram.startTimer({ step: 'screenshot' });
    await callback();
    timer();
  }
}
