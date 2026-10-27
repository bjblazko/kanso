# SPDX-License-Identifier: Apache-2.0
{
  description = "Hüpattl! Kanso — family wiki, notes and more, stored as files";

  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixos-26.05";

  outputs = { self, nixpkgs }:
    let
      systems = [ "aarch64-linux" "x86_64-linux" "aarch64-darwin" ];
      forAllSystems = f: nixpkgs.lib.genAttrs systems (system: f nixpkgs.legacyPackages.${system});
    in
    {
      # One binary with templates and static files embedded. It calls `git`
      # at run time; the service puts git on its PATH.
      packages = forAllSystems (pkgs: rec {
        # go.mod needs Go 1.27.3 (GO-2026-6809 in crypto/x509).
        kanso = (pkgs.buildGoModule.override { go = pkgs.go_1_27; }) {
          pname = "kanso";
          version = self.shortRev or self.dirtyShortRev or "dev";
          src = pkgs.lib.cleanSource self;
          vendorHash = "sha256-Xug6p3J4aSPZqVcbVW42L2TvV9xx4b4M49jsGeMHIRI=";
          subPackages = [ "cmd/kanso" ];
          env.CGO_ENABLED = "0";
          ldflags = [ "-s" "-w" ];
          nativeCheckInputs = [ pkgs.git ];
          meta = {
            description = "Family wiki, notes and more, stored as files in a Git repository";
            license = pkgs.lib.licenses.asl20;
            mainProgram = "kanso";
          };
        };
        default = kanso;
      });
    };
}
