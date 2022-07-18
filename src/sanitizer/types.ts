import * as DOMPurify from 'dompurify';

export enum ConfigType {
  DOMPurify = 'DOMPurify',
}

export const isDOMPurifyConfig = (req: SanitizeRequest): req is SanitizeRequest<ConfigType.DOMPurify> => req.configType === ConfigType.DOMPurify;

const allConfigTypes = Object.values(ConfigType);

export type ConfigTypeToConfig = {
  [ConfigType.DOMPurify]: {
    domPurifyConfig?: DOMPurify.Config;
    allowAllLinksInSvgUseTags?: boolean;
  };
};

export const isSanitizeRequest = (obj: any): obj is SanitizeRequest => {
  return Boolean(obj?.content) && allConfigTypes.includes(obj.configType) && typeof obj.config === 'object';
};

export type SanitizeRequest<configType extends ConfigType = ConfigType> = {
  content: Buffer;
  configType: configType;
  config: ConfigTypeToConfig[configType];
};

export type SanitizeResponse = {
  sanitized: Buffer;
};
