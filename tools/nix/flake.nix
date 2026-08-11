{
  description = "default";

  inputs = {
    nixpkgs.url = "github:nixos/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";

    # Format the repo with nix-treefmt.
    treefmt-nix = {
      url = "github:numtide/treefmt-nix";
      inputs.nixpkgs.follows = "nixpkgs";
    };
  };

  outputs =
    {
      nixpkgs,
      flake-utils,
      treefmt-nix,
      ...
    }:

    let
      defineOutput =
        system:
        let
          pkgs = nixpkgs.legacyPackages.${system};

          treefmtEval = treefmt-nix.lib.evalModule pkgs ./treefmt.nix;

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
          formatter = treefmtEval.config.build.wrapper;

          packages.treefmt = treefmtEval.config.build.wrapper;

          devShells = {
            default = pkgs.mkShell {
              packages = packagesBasic ++ [ treefmtEval.config.build.wrapper ];
            };
          };
        };

    in
    flake-utils.lib.eachDefaultSystem defineOutput;
}
