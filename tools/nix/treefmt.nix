{
  # Used to find the project root
  projectRootFile = ".git/config";

  programs = {
    # Markdown, JSON, YAML, etc.
    prettier.enable = true;

    # Shell.
    shfmt = {
      enable = true;
      indent_size = 4;
    };
    shellcheck.enable = true;

    gofmt.enable = true;
    goimports.enable = true;

    # Nix.
    nixfmt.enable = true;

    # Typos.
    typos.enable = false;
  };
}
