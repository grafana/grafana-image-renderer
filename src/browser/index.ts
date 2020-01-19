import { RenderingConfig } from '../config';
import { Logger } from '../logger';
import { Browser, BrowserTimings, NoOpBrowserTiming } from './browser';
import { ClusteredBrowser } from './clustered';
import { ReusableBrowser } from './reusable';

export function createBrowser(config: RenderingConfig, log: Logger, timings: BrowserTimings): Browser {
  if (config.mode === 'clustered') {
    log.info('using clustered browser', 'mode', config.clustering.mode, 'maxConcurrency', config.clustering.maxConcurrency);
    return new ClusteredBrowser(config, log, timings);
  }

  if (config.mode === 'reusable') {
    log.info('using reusable browser');
    return new ReusableBrowser(config, log, timings);
  }

  return new Browser(config, log, timings);
}

export { Browser, NoOpBrowserTiming };
