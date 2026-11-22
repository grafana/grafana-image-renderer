{
  description = "Grafana Image Renderer and reproducible Chromium container images";

  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";

  outputs =
    { nixpkgs, ... }:
    let
      forLinux = nixpkgs.lib.genAttrs [
        "x86_64-linux"
        "aarch64-linux"
      ];
    in
    {
      packages = forLinux (
        system:
        let
          pkgs = nixpkgs.legacyPackages.${system};
          chromium = (import ./nix/chromium.nix { inherit pkgs; }).browser;
        in
        {
          browser = chromium;
          image = import ./nix/image.nix { inherit pkgs chromium; };
        }
      );
    };
}
