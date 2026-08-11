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

    goimports.enable = true;

    # Nix.
    nixfmt.enable = true;

    # Typos.
    typos.enable = false;
  };

  settings = {
    global.excludes = [
      "vendor/**"
    ];
    formatter.prettier.options = [
      "--prose-wrap"
      "always"
      "--print-width"
      "100"
    ];
  };
}
