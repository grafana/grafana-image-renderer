
export interface Logger {
  info(...optionalParams: any[]);
}


export class ConsoleLogger {

  info(...optionalParams: any[]) {
    console.log(...optionalParams);
  }

}

export class PluginLogger {

  info(...optionalParams: any[]) {
    console.error(...optionalParams);
  }

}
