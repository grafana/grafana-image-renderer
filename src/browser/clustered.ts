import { Cluster } from 'puppeteer-cluster';
import { Browser, RenderResponse, RenderOptions, RenderCSVOptions, RenderCSVResponse } from './browser';
import { Logger } from '../logger';
import { RenderingConfig, ClusteringConfig } from '../config';

enum RenderType {
  CSV = 'csv',
  PNG = 'png',
}

interface ClusterOptions {
  options: RenderOptions | RenderCSVOptions;
  renderType: RenderType;
}

type ClusterResponse = RenderResponse | RenderCSVResponse;

export class ClusteredBrowser extends Browser {
  cluster: Cluster<ClusterOptions, ClusterResponse>;
  clusteringConfig: ClusteringConfig;
  concurrency: number;

  constructor(config: RenderingConfig, log: Logger) {
    super(config, log);

    this.clusteringConfig = config.clustering;
    this.concurrency = Cluster.CONCURRENCY_BROWSER;

    if (this.clusteringConfig.mode === 'context') {
      this.concurrency = Cluster.CONCURRENCY_CONTEXT;
    }
  }

  async start(): Promise<void> {
    const launcherOptions = this.getLauncherOptions({});
    this.cluster = await Cluster.launch({
      concurrency: this.concurrency,
      maxConcurrency: this.clusteringConfig.maxConcurrency,
      puppeteerOptions: launcherOptions,
    });
    await this.cluster.task(async ({ page, data }) => {
      if (data.options.timezone) {
        // set timezone
        await page.emulateTimezone(data.options.timezone);
      }

      try {
        this.addPageListeners(page);
        switch (data.renderType) {
          case RenderType.CSV:
            return await this.exportCSV(page, data.options);
          case RenderType.PNG:
          default:
            return await this.takeScreenshot(page, data.options);
        }
      } finally {
        this.removePageListeners(page);
      }
    });
  }

  async render(options: RenderOptions): Promise<RenderResponse> {
    this.validateImageOptions(options);
    return this.cluster.execute({ options, renderType: RenderType.PNG });
  }

  async renderCSV(options: RenderCSVOptions): Promise<RenderCSVResponse> {
    this.validateRenderOptions(options);
    return this.cluster.execute({ options, renderType: RenderType.CSV });
  }
}
