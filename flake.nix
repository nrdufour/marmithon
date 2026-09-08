{
  description = "marmithon - Simple IRC bot with utility commands";

  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";

  outputs =
    { self, nixpkgs }:
    let
      version = "0.1.0";

      # Only the platforms anything is built for. Adding a system is cheap;
      # claiming one that was never built is not.
      systems = [
        "x86_64-linux"
        "aarch64-linux"
      ];

      forAllSystems = f: nixpkgs.lib.genAttrs systems (system: f nixpkgs.legacyPackages.${system});
    in
    {
      packages = forAllSystems (pkgs: {
        default = (pkgs.buildGoModule.override { go = pkgs.go_1_27; }) {
          pname = "marmithon";
          inherit version;
          src = ./.;

          vendorHash = "sha256-pCbMEYGzgbU3t8SRBzPa74cemeQtZEjMBLlWzbtR4mA=";

          # test_nmi/ is a throwaway main package; only the bot ships.
          subPackages = [ "." ];

          # Static binary that cross-compiles: modernc.org/sqlite is pure Go.
          env.CGO_ENABLED = 0;

          ldflags = [
            "-s"
            "-w"
            "-X marmithon/command.GitCommit=${self.shortRev or self.dirtyShortRev or "unknown"}"
            "-X marmithon/command.BuildTime=${self.lastModifiedDate or "unknown"}"
          ];

          meta = {
            description = "Simple IRC bot with utility commands";
            mainProgram = "marmithon";
            license = pkgs.lib.licenses.mit;
            platforms = systems;
          };
        };
      });

      devShells = forAllSystems (pkgs: {
        default = pkgs.mkShell {
          packages = with pkgs; [
            go_1_27 # matches the toolchain directive in go.mod
            gopls
            gotools # goimports
            # staticcheck carries its own go/types, so it has to be compiled
            # with the same Go it analyses or it chokes on the 1.27 stdlib.
            (go-tools.override { buildGoModule = buildGoModule.override { go = go_1_27; }; })
            sqlite # poke at seen.db
            flyctl
            just # the task runner; `just` alone lists every recipe
            tea # forge.internal PRs and issues (see the create-pr skill)
            forgejo-cli # `fj`: same forge, the other CLI
            nixfmt
          ];

          # Matches how the binary is built, so a local `go test` cannot pass
          # under settings the package never uses.
          env.CGO_ENABLED = "0";
        };
      });

      formatter = forAllSystems (pkgs: pkgs.nixfmt);
    };
}
