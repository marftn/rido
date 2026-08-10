{
  description = "default";

  inputs = {
    nixpkgs.url = "github:nixos/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = {nixpkgs, flake-utils, ...}:


    let defineOutput =
      system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
        packagesBasic = with pkgs; [
          # Shells
          fish

          # Essentials.
          git
          just

          # Main languages.
          go_1_26

          # Linting and LSP and debuggers.
          gopls
          golines
          gotools
          golangci-lint
          golangci-lint-langserver
          typos-lsp
        ];

      in
      {
        devShells = {
          default = pkgs.mkShell {
            packages = packagesBasic;
          };
        };
      };

  in
  flake-utils.lib.eachDefaultSystem defineOutput;
}
