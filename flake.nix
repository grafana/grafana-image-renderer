{
  description = "Grafana Image Renderer and reproducible Chromium container images";

  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";

  outputs =
    { nixpkgs, ... }:
    {
      packages =
        nixpkgs.lib.genAttrs
          [
            "x86_64-linux"
            "aarch64-linux"
          ]
          (
            system:
            let
              pkgs = import nixpkgs { inherit system; };
              image = chromium: import ./nix/image.nix { inherit pkgs chromium; };
            in
            {
              image = image pkgs.chromium.browser;
              image-custom = image (import ./nix/chromium.nix { inherit pkgs; });
              update-chromium = import ./nix/update-chromium.nix { inherit pkgs; };
            }
          );
    };
}
