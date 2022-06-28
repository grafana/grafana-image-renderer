export interface LogWriter {
  write(message, encoding);
}

export interface Logger {
  errorWriter: LogWriter;
  debugWriter: LogWriter;
  debug(message?: string, ...optionalParams: any[]);
  info(message?: string, ...optionalParams: any[]);
  warn(message?: string, ...optionalParams: any[]);
  error(message?: string, ...optionalParams: any[]);
}

export class PluginLogger implements Logger {
  errorWriter: LogWriter;
  debugWriter: LogWriter;

  private logEntry(level: string, message?: string, ...optionalParams: any[]) {
    const logEntry = {
      '@level': level,
    };

    if (message) {
      logEntry['@message'] = message;
    }

    if (optionalParams) {
      for (let n = 0; n < optionalParams.length; n += 2) {
        const key = optionalParams[n];
        const value = optionalParams[n + 1];

        if (key !== null && value !== null) {
          logEntry[key] = value;
        }
      }
    }

    console.error(JSON.stringify(logEntry));
  }

  debug(message?: string, ...optionalParams: any[]) {
    this.logEntry('debug', message, ...optionalParams);
  }

  info(message?: string, ...optionalParams: any[]) {
    this.logEntry('info', message, ...optionalParams);
  }

  warn(message?: string, ...optionalParams: any[]) {
    this.logEntry('warn', message, ...optionalParams);
  }

  error(message?: string, ...optionalParams: any[]) {
    this.logEntry('error', message, ...optionalParams);
  }
}
