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
