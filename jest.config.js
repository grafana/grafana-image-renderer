/** @type {import('ts-jest').JestConfigWithTsJest} */
module.exports = {
  preset: 'ts-jest',
  testEnvironment: 'node',
  reporters: ['<rootDir>/tests/reporter.js'],
  testTimeout: 20000,
};