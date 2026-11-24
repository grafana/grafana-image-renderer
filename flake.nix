{
  description = "Grafana Image Renderer and reproducible Chromium container images";

  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";

  outputs =
    { nixpkgs, ... }:
    {
      packages = nixpkgs.lib.genAttrs [
        "x86_64-linux"
        "aarch64-linux"
      ] (system: {
        image = import ./nix/image.nix {
          pkgs = nixpkgs.legacyPackages.${system};
        };
      });
    };
}
