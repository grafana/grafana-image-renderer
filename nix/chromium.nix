{ pkgs }:

let
  recipe = import ./chromium-recipe.nix { inherit pkgs; };
in
(recipe.chromium.override {
  upstream-info = (pkgs.lib.importJSON ./chromium-info.json).chromium;
}).browser
