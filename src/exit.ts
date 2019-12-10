import { Logger } from './logger';

type cleanUpFn = () => void;
const cleanUpHandlers: cleanUpFn[] = [];

export function registerExitCleanUp(fn) {
  cleanUpHandlers.push(fn);
}

export class ExitManager {
  constructor(private log: Logger) {
    process.stdin.resume(); //so the program will not close instantly

    //do something when app is closing
    process.on('exit', this.exitHandler.bind(this));

    //catches ctrl+c event
    process.on('SIGINT', this.exitHandler.bind(this));

    // catches "kill pid" (for example: nodemon restart)
    process.on('SIGUSR1', this.exitHandler.bind(this));
    process.on('SIGUSR2', this.exitHandler.bind(this));

    //catches uncaught exceptions
    process.on('uncaughtException', this.exitHandler.bind(this));
  }

  exitHandler(options, err) {
    for (const fn of cleanUpHandlers) {
      try {
        fn();
      } catch (err) {
        this.log.info('Failed to call cleanup function ' + err);
      }
    }
    if (err) {
      this.log.info(err);
    }
    process.exit();
  }
}
