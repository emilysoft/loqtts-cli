{
  description = "loqtts - CLI and Discord bot for the Loquendo TTS 7 SDK";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";

    # Proprietary Loquendo TTS 7 SDK. Point this at your install, e.g.:
    #   nix build . --override-input loquendo-sdk path:/path/to/LTTS7
    # Only include/ and lib/ are needed to build the CLI.
    loquendo-sdk = {
      url = "/home/nit/Games/loquendo/drive_c/Program Files (x86)/Loquendo/LTTS7";
      flake = false;
    };
  };

  outputs = {
    self,
    nixpkgs,
    loquendo-sdk,
  }: let
    system = "x86_64-linux";
    pkgs = nixpkgs.legacyPackages.${system};
    mingw = pkgs.pkgsCross.mingw32;
  in {
    packages.${system} = {
      # Cross-compiled loqtts.exe for Windows (+ the in-memory audio DLL).
      default = pkgs.stdenv.mkDerivation {
        pname = "loqtts";
        version = "1.0.0";
        src = self;

        sdk = loquendo-sdk;

        nativeBuildInputs = [
          mingw.stdenv.cc # i686-w64-mingw32-gcc + dlltool
          pkgs.gnumake
        ];

        buildPhase = ''
          runHook preBuild
          make ROOT="$sdk"
          runHook postBuild
        '';

        installPhase = ''
          runHook preInstall
          mkdir -p $out/bin
          cp loqtts.exe LoqTTS7AudioUser.dll $out/bin/
          runHook postInstall
        '';

        meta = {
          description = "Command line interface for the Loquendo TTS 7 SDK";
          license = pkgs.lib.licenses.unfree;
          platforms = ["x86_64-linux"];
        };
      };

      # The Discord bot (Go, native Linux binary).
      bot = pkgs.buildGoModule {
        pname = "loqttsbot";
        version = "0.1.0";
        src = ./bot;
        subPackages = ["./cmd/loqttsbot"];

        nativeBuildInputs = [
          pkgs.pkg-config
          pkgs.makeWrapper
        ];
        buildInputs = [
          pkgs.libopus
          pkgs.opusfile
          pkgs.libogg
        ];
        vendorHash = "sha256-NEPgNoRD8DogpAx7l6YWuC24Dcb73oHxiO2lbCTZpcI=";

        postInstall = ''
          wrapProgram $out/bin/loqttsbot \
            --prefix LD_LIBRARY_PATH : ${pkgs.lib.makeLibraryPath [ pkgs.libopus pkgs.opusfile pkgs.libogg ]} \
            --prefix PATH : ${pkgs.lib.makeBinPath [ pkgs.steam-run ]}
        '';

        meta = {
          description = "Discord bot that speaks with Loquendo TTS 7";
          license = pkgs.lib.licenses.unfree;
          platforms = [ "x86_64-linux" ];
        };
      };
    };

    # Development shell: cross toolchain + helpers to build, deploy and run.
    devShells.${system} = {
      default = pkgs.mkShell {
        name = "loqtts-dev";

        packages = [
          mingw.stdenv.cc
          pkgs.gnumake
          pkgs.steam-run # lets the .exe run under Wine on NixOS
          pkgs.go
          pkgs.pkg-config
          pkgs.libopus
          pkgs.opusfile
          pkgs.libogg
          (pkgs.writeShellScriptBin "loqtts-build" ''
            exec make ROOT="''${LTTS7_ROOT:?set LTTS7_ROOT to your Loquendo SDK path}" "$@"
          '')
          (pkgs.writeShellScriptBin "loqtts-deploy" ''
            set -e
            install=''${LOQTTS_INSTALL:?set LOQTTS_INSTALL to your Loquendo install path}
            root=''${PWD}
            while [ "$root" != "/" ] && [ ! -f "$root/loqtts.exe" ]; do root="$(dirname "$root")"; done
            [ -f "$root/loqtts.exe" ] || { echo "loqtts.exe not found (run loqtts-build first)"; exit 1; }
            for f in loqtts.exe LoqTTS7AudioUser.dll; do
              [ -f "$root/$f" ] || { echo "missing $root/$f (run loqtts-build first)"; exit 1; }
              cp "$root/$f" "$install/bin/"
            done
            echo "deployed loqtts.exe + LoqTTS7AudioUser.dll to $install/bin"
          '')
          (pkgs.writeShellScriptBin "loqtts-run" ''
            set -e
            install=''${LOQTTS_INSTALL:?set LOQTTS_INSTALL to your Loquendo install path}
            root=''${PWD}
            while [ "$root" != "/" ] && [ ! -f "$root/loqtts.exe" ]; do root="$(dirname "$root")"; done
            [ -f "$root/loqtts.exe" ] && cp "$root/loqtts.exe" "$install/bin/loqtts.exe"
            [ -f "$root/LoqTTS7AudioUser.dll" ] && cp "$root/LoqTTS7AudioUser.dll" "$install/bin/LoqTTS7AudioUser.dll"
            cd "$install/bin"
            wine64=''${WINE64:-$HOME/.local/share/lutris/runners/wine/wine-ge-8-26-x86_64/bin/wine64}
            exec steam-run env WINEPREFIX=''${WINEPREFIX:-$HOME/Games/loquendo} "$wine64" ./loqtts.exe "$@"
          '')
          (pkgs.writeShellScriptBin "loqtts-bot" ''
            set -e
            [ -d bot ] && cd bot
            bin="$(mktemp -d)/loqttsbot"
            go build -o "$bin" ./cmd/loqttsbot
            exec "$bin" "$@"
          '')
        ];

        env = {
          LTTS7_ROOT = toString loquendo-sdk; # build: include/lib from the flake input
          LOQTTS_INSTALL = "/home/nit/Games/loquendo/drive_c/Program Files (x86)/Loquendo/LTTS7"; # run: DLLs resolve by registered absolute paths
          WINEPREFIX = "/home/nit/Games/loquendo";
          LD_LIBRARY_PATH = pkgs.lib.makeLibraryPath [pkgs.libopus pkgs.opusfile pkgs.libogg];
        };

        shellHook = ''
          echo "loqtts dev shell:"
          echo "  loqtts-build [targets]      build loqtts.exe + LoqTTS7AudioUser.dll"
          echo "  loqtts-deploy               copy exe + dll to LOQTTS_INSTALL/bin"
          echo "  loqtts-run [args]           run ./loqtts.exe under Wine (steam-run)"
          echo "  loqtts-bot [args]           build and run the Discord bot"
          echo "  LTTS7_ROOT=$LTTS7_ROOT"
        '';
      };
    };
  };
}
