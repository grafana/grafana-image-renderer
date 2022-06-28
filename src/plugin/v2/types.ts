import { ConfigType } from '../../types';

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
}

export interface GRPCSanitizeResponse {
  error: string;
  sanitized: Buffer;
}
