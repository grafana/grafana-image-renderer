import { Cluster as PoolpeteerCluster } from 'poolpeteer';
import { Cluster as PuppeteerCluster } from 'puppeteer-cluster';
import { ImageRenderOptions, RenderOptions } from '../types';
import { Browser, RenderResponse, RenderCSVResponse, Metrics } from './browser';
import { Logger } from '../logger';
import { RenderingConfig, ClusteringConfig } from '../config/rendering';

enum RenderType {
  CSV = 'csv',
  PNG = 'png',
}

interface ClusterOptions {
  groupId?: string;
  options: RenderOptions | ImageRenderOptions;
  renderType: RenderType;
  signal: AbortSignal;
}

type ClusterResponse = RenderResponse | RenderCSVResponse;

const contextPerRenderKey = 'contextPerRenderKey';

type Cluster<JobData = any, ReturnData = any> = PuppeteerCluster<JobData, ReturnData> | PoolpeteerCluster<JobData, ReturnData>;

export class ClusteredBrowser extends Browser {
  cluster: Cluster<ClusterOptions, ClusterResponse>;
  clusteringConfig: ClusteringConfig;
  concurrency: number;

  constructor(config: RenderingConfig, log: Logger, metrics: Metrics) {
    super(config, log, metrics);

    this.clusteringConfig = config.clustering;
    this.concurrency = PuppeteerCluster.CONCURRENCY_BROWSER;

    if (this.clusteringConfig.mode === 'context') {
      this.concurrency = PuppeteerCluster.CONCURRENCY_CONTEXT;
    }

    if (this.clusteringConfig.mode === contextPerRenderKey) {
      this.concurrency = PoolpeteerCluster.CONCURRENCY_CONTEXT_PER_REQUEST_GROUP;
    }
  }

  shouldUsePoolpeteer(): boolean {
    return this.clusteringConfig.mode === contextPerRenderKey;
  }

  async createCluster(): Promise<Cluster<ClusterOptions, ClusterResponse>> {
    const launcherOptions = this.getLauncherOptions({});

    const clusterOptions = {
      concurrency: this.concurrency,
      workerShutdownTimeout: 5000,
      monitor: this.clusteringConfig.monitor,
      maxConcurrency: this.clusteringConfig.maxConcurrency,
      timeout: this.clusteringConfig.timeout * 1000,
      puppeteerOptions: launcherOptions,
    };

    // TODO use poolpeteer by default after initial release and testing (8.5?)
    if (this.shouldUsePoolpeteer()) {
      this.log.debug('Launching Browser cluster with poolpeteer');
      return PoolpeteerCluster.launch(clusterOptions);
    }

    this.log.debug('Launching Browser cluster with puppeteer-cluster');
    return PuppeteerCluster.launch(clusterOptions);
  }

  async start(): Promise<void> {
    this.cluster = await this.createCluster();
    await this.cluster.task(async ({ page, data }) => {
      const { options, renderType, signal } = data;
      if (options.timezone) {
        // set timezone
        await page.emulateTimezone(options.timezone);
      }

      try {
        await this.addPageListeners(page, options.headers);
        switch (renderType) {
          case RenderType.CSV:
            return await this.exportCSV(page, options, signal);
          case RenderType.PNG:
          default:
            return await this.takeScreenshot(page, options as ImageRenderOptions, signal);
        }
      } finally {
        this.removePageListeners(page);
      }
    });
  }

  private getGroupId = (options: ImageRenderOptions | RenderOptions) => {
    if (this.clusteringConfig.mode === contextPerRenderKey) {
      return `${options.domain}${options.renderKey}`;
    }

    return undefined;
  };

  async render(options: ImageRenderOptions, signal: AbortSignal): Promise<RenderResponse> {
    this.validateImageOptions(options);
    return this.cluster.execute({ groupId: this.getGroupId(options), options, renderType: RenderType.PNG, signal });
  }

  async renderCSV(options: RenderOptions, signal: AbortSignal): Promise<RenderCSVResponse> {
    this.validateRenderOptions(options);
    return this.cluster.execute({ groupId: this.getGroupId(options), options, renderType: RenderType.CSV, signal });
  }
}
