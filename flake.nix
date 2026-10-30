{
  description = "homelab-images development environment";

  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";

  outputs =
    { nixpkgs, ... }:
    let
      systems = [
        "x86_64-linux"
        "aarch64-linux"
        "x86_64-darwin"
        "aarch64-darwin"
      ];
      forAllSystems = nixpkgs.lib.genAttrs systems;
    in
    {
      devShells = forAllSystems (
        system:
        let
          pkgs = nixpkgs.legacyPackages.${system};

          # not packaged in nixpkgs
          controller-gen = pkgs.buildGoModule rec {
            pname = "controller-gen";
            # renovate: datasource=github-releases depName=kubernetes-sigs/controller-tools extractVersion=^v(?<version>.+)$
            version = "0.23.0";
            src = pkgs.fetchFromGitHub {
              owner = "kubernetes-sigs";
              repo = "controller-tools";
              rev = "v${version}";
              hash = "sha256-BvFMLjxC5iZk9gmTgMA6ZFBHXwGzi35sz63uVB0A+xs=";
            };
            vendorHash = "sha256-eGp5KRangcOwvAEFDJrKBFZGLqB4tqK2AiltFlG4hM0=";
            subPackages = [ "cmd/controller-gen" ];
            doCheck = false;
          };
        in
        {
          default = pkgs.mkShell {
            packages = with pkgs; [
              controller-gen
              gh
              git
              go_1_27
              kubernetes-helm
              nixfmt
              prettier
            ];
          };
        }
      );
    };
}
