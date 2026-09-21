{
  description = "rido - move files into a store and leave symlinks behind";

  inputs = {
    nixpkgs.url = "github:nixos/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs =
    {
      self,
      nixpkgs,
      flake-utils,
    }:
    flake-utils.lib.eachDefaultSystem (
      system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
      in
      {
        packages.default = pkgs.buildGoModule {
          pname = "rido";
          version = self.shortRev or "dev";
          src = ./.;

          vendorHash = "sha256-4g8b8uHGe9TZnM3SlmBFnCI5GFQaowJuVOnvHK+vcZQ=";

          meta = {
            description = "Move files or directories into a store, leave symlinks.";
            homepage = "https://github.com/marftn/rido";
            license = pkgs.lib.licenses.mit;
            mainProgram = "rido";
          };
        };
      }
    );
}
