{
  pkgs,
  infoFile ? ./chromium-info.json,
}:

(pkgs.chromium.override {
  cupsSupport = false;
  upstream-info = (pkgs.lib.importJSON infoFile).chromium;
}).browser
