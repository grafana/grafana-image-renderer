{
  pkgs,
  infoFile ? ./chromium-info.json,
}:

let
  recipe = import ./chromium-recipe.nix { inherit pkgs; };
in
(recipe.chromium.override {
  cupsSupport = false;
  upstream-info = (pkgs.lib.importJSON infoFile).chromium;
}).browser
