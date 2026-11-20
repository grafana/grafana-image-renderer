{ pkgs }:

let
  # Paths inside these patches are relative to Nixpkgs' Chromium recipe directory.
  # https://github.com/NixOS/nixpkgs/tree/master/pkgs/applications/networking/browsers/chromium
  patches = [ ];

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
