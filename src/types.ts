import * as DOMPurify from 'dompurify';

export interface HTTPHeaders {
  'Accept-Language'?: string;
  [header: string]: string | undefined;
}

// Common options for CSV and Images
export interface RenderOptions {
  url: string;
  filePath: string;
  timeout: number; // seconds
  renderKey: string;
  domain: string;
  timezone?: string;
  encoding?: string;
  headers?: HTTPHeaders;
}

export interface ImageRenderOptions extends RenderOptions {
  width: string | number;
  height: string | number;
  deviceScaleFactor?: string | number;
  scrollDelay?: number;

  // Runtime options derived from the input
  fullPageImage?: boolean;
  scaleImage?: number;
}

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
