{ pkgs ? import <nixpkgs> {
    config.allowUnfree = true;
  }
, unstable ? import (fetchTarball "https://github.com/NixOS/nixpkgs/archive/nixos-unstable.tar.gz") {
    config.allowUnfree = true;
  }
}:
  pkgs.mkShell {
    # nativeBuildInputs is usually what you want -- tools you need to run
    nativeBuildInputs = with pkgs.buildPackages; [
      flyctl
      go-task
      unstable.claude-code
      jq
      sqlite
    ];
}
