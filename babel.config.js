module.exports = {
  presets: [
    [
      '@babel/preset-env',
      {
        "bugfixes": true,
        "browserslistEnv": "dev",
        "useBuiltIns": "entry",
        "corejs": "3.10"
      }
    ],
    [
      "@babel/preset-typescript",
      {
        "allowNamespaces": true,
        "allowDeclareFields": true
      }
    ],
  ],
};