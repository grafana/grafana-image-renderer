import { RenderType } from '../../browser/browser';

export interface StringList {
  values: string[];
}

export interface RenderRequest {
  renderType: RenderType;
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
}

export interface RenderResponse {
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
