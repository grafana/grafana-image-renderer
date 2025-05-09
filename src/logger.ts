import * as winston from 'winston';
import { LoggingConfig } from './service/config';
import { context, trace } from '@opentelemetry/api';

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

export class ConsoleLogger implements Logger {
  errorWriter: LogWriter;
  debugWriter: LogWriter;
  logger: winston.Logger;

  constructor(config: LoggingConfig) {
    const transports: any[] = [];

    if (config.console) {
      const options: any = {
        exitOnError: false,
      };
      if (config.console.level) {
        options.level = config.console.level;
      }
      const formatters: any[] = [];
      if (config.console.colorize) {
        formatters.push(winston.format.colorize());
      }

      if (config.console.json) {
        formatters.push(winston.format.json());
      } else {
        formatters.push(winston.format.align());
        formatters.push(winston.format.simple());
      }

      options.format = winston.format.combine(...(formatters as any));
      transports.push(new winston.transports.Console(options));
    }

    //@opentelemetry/instrumentation-winston auto inject trace-context into Winston log records

    this.logger = winston.createLogger({
      level: config.level,
      exitOnError: false,
      transports: transports,
    });

    this.errorWriter = {
      write: (message) => {
        this.logger.error(message);
      },
    };
    this.debugWriter = {
      write: (message) => {
        this.logger.debug(message);
      },
    };
  }

  private logEntry(level: string, message: string, ...optionalParams: any[]) {
    const meta: any = {};
    if (optionalParams) {
      for (let n = 0; n < optionalParams.length; n += 2) {
        const key = optionalParams[n];
        const value = optionalParams[n + 1];

        if (key !== null && value !== null) {
          meta[key] = value;
        }
      }
    }

    this.logger.log(level, message, meta);
  }

  debug(message: string, ...optionalParams: any[]) {
    this.logEntry('debug', message, ...optionalParams);
  }

  info(message: string, ...optionalParams: any[]) {
    this.logEntry('info', message, ...optionalParams);
  }

  warn(message: string, ...optionalParams: any[]) {
    this.logEntry('warn', message, ...optionalParams);
  }

  error(message: string, ...optionalParams: any[]) {
    this.logEntry('error', message, ...optionalParams);
  }
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
