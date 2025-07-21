const eslint = require('@eslint/js');
const tseslint = require('typescript-eslint');
const grafanaConfig = require('@grafana/eslint-config/flat');

module.exports = tseslint.config(
  {
    ignores: [
      ".github",
      ".yarn",
      "**/build/",
      "**/compiled/",
      "**/dist/",
      "node_modules/",
      "scripts/",
      "devenv/",
      ".prettierrc.js",
      "*.config.js",
    ],
  },

  eslint.configs.recommended,
  tseslint.configs.recommended,
  grafanaConfig,

  {
    rules: {
      '@typescript-eslint/no-explicit-any': 'off', // Express litters them everywhere.
    },
  },
);
