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

        # Bump this on each release, Nix has no view of the git tag.
        version = "0.1.1";
      in
      {
        packages.default = pkgs.buildGoModule {
          pname = "rido";
          inherit version;
          src = ./.;

          # The flake source carries no .git, so debug.ReadBuildInfo() would
          # report "(devel)". Stamp the version explicitly instead.
          ldflags = [ "-X main.buildVersion=v${version}+${self.shortRev or "dirty"}" ];

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
