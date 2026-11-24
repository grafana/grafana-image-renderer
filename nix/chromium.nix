{ pkgs }:

let
  info = (pkgs.lib.importJSON ./chromium-info.json).chromium or null;
in
if info != null && pkgs.lib.versionOlder pkgs.chromium.version info.version then
  pkgs.chromium.override { upstream-info = info; }
else
  pkgs.chromium
