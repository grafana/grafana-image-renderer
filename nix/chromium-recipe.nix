{ pkgs }:

let
  patches = import ./chromium-recipe-patches.nix;
  source = pkgs.path + "/pkgs/applications/networking/browsers/chromium";
  patchedSource = pkgs.applyPatches {
    name = "image-renderer-chromium-recipe";
    src = source;
    inherit patches;
  };
in
{
  source = if patches == [ ] then source else patchedSource;
  chromium =
    if patches == [ ] then pkgs.chromium else pkgs.callPackage (patchedSource + "/default.nix") { };
}
