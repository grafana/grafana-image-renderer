import * as fs from 'fs';
import * as path from 'path';
import * as cron from 'node-cron';
import { Logger } from '../logger';
import { FileRetentionConfig } from '../config';

export class CleanupJob {
  logger: Logger;
  config: FileRetentionConfig;

  constructor(logger: Logger, config: FileRetentionConfig) {
    this.logger = logger;
    this.config = config;
  }

  run() {
    if (!this.config.enabled) {
      this.logger.debug('Cleanup job disabled');
      return;
    }

    if (!cron.validate(this.config.cronSchedule)) {
      this.logger.warn('Invalid cron schedule expression, using default');
      this.config.cronSchedule = '0 0 * * *';
    }

    cron.schedule(this.config.cronSchedule, () => {
      this.logger.debug('Running cleanup job...');

      const filesToDelete = fs
        .readdirSync(this.config.tempDir, { withFileTypes: true })
        .filter(this.shouldDeleteFile)
        .map(item => item.name);

      let deletedFileCount = 0;
      if (filesToDelete.length > 0) {
        for (let n = 0; n < filesToDelete.length; n++) {
          const file = path.join(this.config.tempDir, filesToDelete[n]);
          try {
            fs.unlinkSync(file);
            deletedFileCount++;
          } catch (err) {
            this.logger.error('Failed to delete file', 'file', file, 'error', err);
          }
        }
      }

      this.logger.debug('Cleanup job done', 'deleted', deletedFileCount);
    });

    let retention = `${this.config.retentionSeconds}s`;
    if (this.config.retentionSeconds >= 3600) {
      retention = `${this.config.retentionSeconds / 3600}h`;
    } else if (this.config.retentionSeconds >= 60) {
      retention = `${this.config.retentionSeconds / 60}m`;
    }

    this.logger.debug('Cleanup job scheduled', 'tempDir', this.config.tempDir, 'retention', retention, 'cronSchedule', this.config.cronSchedule);
  }

  shouldDeleteFile = (item: fs.Dirent): boolean => {
    if (item.isDirectory()) {
      return false;
    }

    if (path.extname(item.name) !== '.png') {
      return false;
    }

    const s = fs.statSync(path.join(this.config.tempDir, item.name));
    const fileTimeMs = s.mtime.valueOf() + this.config.retentionSeconds * 1000;

    return fileTimeMs < new Date().valueOf();
  };
}
