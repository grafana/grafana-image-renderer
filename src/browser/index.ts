import { RenderingConfig } from '../config';
import { Logger } from '../logger';
import { Browser, Metrics } from './browser';
import { ClusteredBrowser } from './clustered';
import { ReusableBrowser } from './reusable';

export function createBrowser(config: RenderingConfig, log: Logger, metrics: Metrics): Browser {
  if (config.mode === 'clustered') {
    log.info('using clustered browser', 'mode', config.clustering.mode, 'maxConcurrency', config.clustering.maxConcurrency);
    return new ClusteredBrowser(config, log, metrics);
  }

  if (config.mode === 'reusable') {
    log.info('using reusable browser');
    return new ReusableBrowser(config, log, metrics);
  }

  return new Browser(config, log, metrics);
}

export { Browser };
