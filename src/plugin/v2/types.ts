import { ConfigType } from '../../sanitizer/types';

export interface StringList {
  values: string[];
}

export interface RenderRequest {
  url: string;
  width: number;
  height: number;
  deviceScaleFactor: number;
  filePath: string;
  renderKey: string;
  domain: string;
  timeout: number;
  timezone: string;
  headers: {
    [header: string]: StringList;
  };
  authToken: string;
  encoding: string;
}

export interface RenderResponse {
  error?: any;
}

export interface RenderCSVRequest {
  url: string;
  filePath: string;
  renderKey: string;
  domain: string;
  timeout: number;
  timezone: string;
  headers: {
    [header: string]: StringList;
  };
  authToken: string;
}

export interface RenderCSVResponse {
  error?: any;
  fileName?: string;
}

export interface CollectMetricsRequest {}

export interface MetricsPayload {
  prometheus: Buffer;
}

export interface CollectMetricsResponse {
  metrics: MetricsPayload;
}

export interface CheckHealthRequest {
  config: any;
}

export enum HealthStatus {
  UNKNOWN = 0,
  OK = 1,
  ERROR = 2,
}

export interface CheckHealthResponse {
  status: HealthStatus;
  message?: string;
  jsonDetails?: Buffer;
}

export interface GRPCSanitizeRequest {
  filename: string;
  content: Buffer;
  configType: ConfigType;
  config: Buffer;
  allowAllLinksInSvgUseTags: boolean;
  authToken: string;
}

export interface GRPCSanitizeResponse {
  error: string;
  sanitized: Buffer;
}
