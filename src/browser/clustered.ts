import { Cluster } from 'puppeteer-cluster';
import { Browser, RenderResponse, RenderOptions } from './browser';
import { Logger } from '../logger';
import { RenderingConfig, ClusteringConfig } from '../config';
import CancellationToken from 'cancellationtoken';

export class ClusteredBrowser extends Browser {
  cluster: Cluster<any, RenderResponse>;
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
      if (data.timezone) {
        // set timezone
        await page.emulateTimezone(data.timezone);
      }

      try {
        this.addPageListeners(page);
        return await this.takeScreenshot(page, data);
      } finally {
        this.removePageListeners(page);
      }
    });
  }

  async render(token: CancellationToken, options: RenderOptions): Promise<RenderResponse> {
    this.validateOptions(options);
    return await this.cluster.execute(options);
  }
}
