import {defaultServiceConfig, populateServiceConfigFromEnv, ServiceConfig} from "./service/config";
import * as minimist from 'minimist';
const fs = require("fs");
const _ = require("lodash");

function readJSONFileSync(filePath: string) {
    const rawdata = fs.readFileSync(filePath, 'utf8');
    return JSON.parse(rawdata);
};

export const getServiceConfig = (): ServiceConfig => {
    let config: ServiceConfig = defaultServiceConfig;

    const argv = minimist(process.argv.slice(2));
    if (argv.config) {
        try {
            const fileConfig = readJSONFileSync(argv.config);
            config = _.merge(config, fileConfig);
        } catch (e) {
            console.error('failed to read config from path', argv.config, 'error', e);
        }
    }
    const env = Object.assign({}, process.env);
    populateServiceConfigFromEnv(config, env);
    return config;
}


