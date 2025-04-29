import * as fs from 'fs';
import * as _ from 'lodash';
import * as minimist from 'minimist';
import { defaultServiceConfig, populateServiceConfigFromEnv, ServiceConfig } from '../service/config';
import { defaultPluginConfig, populatePluginConfigFromEnv, PluginConfig } from '../plugin/v2/config';

export function getConfig(): PluginConfig | ServiceConfig {
  const argv = minimist(process.argv.slice(2));
  const env = Object.assign({}, process.env);
  const command = argv._[0];

  if (command === 'server') {
    let config: ServiceConfig = defaultServiceConfig;

    if (argv.config) {
      try {
        const fileConfig = readJSONFileSync(argv.config);
        config = _.merge(config, fileConfig);
      } catch (e) {
        console.error('failed to read config from path', argv.config, 'error', e);
      }
    }

    populateServiceConfigFromEnv(config, env);

    return config;
  }

  const config: PluginConfig = defaultPluginConfig;
  populatePluginConfigFromEnv(config, env);
  return config;
}

function readJSONFileSync(filePath: string) {
  const rawdata = fs.readFileSync(filePath, 'utf8');
  return JSON.parse(rawdata);
}
