export class TimeoutError extends Error {
  constructor(step) {
    super('Timeout error while performing step: ' + step);
    this.name = 'TimeoutError';
  }
}
