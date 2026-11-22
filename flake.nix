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
          chromium = import ./nix/chromium.nix { inherit pkgs; };
        in
        {
          browser = chromium.browser // {
            isCustom = chromium.version != pkgs.chromium.version;
          };
          image = import ./nix/image.nix {
            inherit pkgs;
            chromium = chromium.browser;
          };
          update-chromium = import ./nix/update-chromium.nix { inherit pkgs; };
        }
      );

      checks = forLinux (system: {
        chromium-selection = import ./nix/chromium-selection-test.nix {
          pkgs = nixpkgs.legacyPackages.${system};
        };
      });
    };
}
