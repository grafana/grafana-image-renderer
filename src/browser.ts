import * as os from 'os';
import * as puppeteer from 'puppeteer';
import { Logger } from './logger';
import { registerExitCleanUp } from './exit';
import uniqueFilename = require('unique-filename');

export class Browser {
  private instance;

  constructor(private log: Logger) {
  }

  async start() {
    this.log.info("Starting chrome");
    this.instance = await puppeteer.launch();

    registerExitCleanUp(() => {
      this.instance.close();
    });
  }

  async render() {
    const page = await this.instance.newPage();
    await page.goto('http://google.com');

    const imagePath = uniqueFilename(os.tmpdir()) + '.png';
    await page.screenshot({path: imagePath});

    return { imagePath: imagePath };
  }
}
