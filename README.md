# loqtts - Loquendo TTS 7 command line interface

A small C command line program for the Loquendo TTS 7 SDK. It lets you use
any installed voice from the terminal: speak text through the audio board,
or synthesize speech to a WAV/raw PCM file, with control over speed, pitch,
volume and voice/language/style selection.

This repository contains only the CLI sources, not the Loquendo TTS SDK.
You need a copy of the Loquendo TTS 7 installation (its `include/`, `lib/`
and `bin/` folders) to build and run it.

## Layout

The folder is designed to be dropped straight into the LTTS7 install
directory as `LTTS7/cli/` (the build scripts default to `..` for the SDK
root). It can also be cloned anywhere and pointed at an LTTS7 install with
`make ROOT=<path>` / `build.bat <path>`.

## Quick start

From a Windows command prompt, with the Loquendo install directory as the
current directory:

```
loqtts "Hello world"
loqtts -v Carmen -o hello.wav "Hola, ¿cómo estás?"
echo "Texto desde stdin" | loqtts
```

Run `loqtts --help` for the full option list.

## Building

### On Windows with Microsoft Visual C++ (recommended)

Open a **Visual Studio Command Prompt** (or a developer PowerShell) in this
directory and run:

```
build.bat
```

This compiles `loqtts.exe` against the SDK import library
`..\lib\LoqTTS7.lib` and the headers in `..\include`. To build against an
LTTS7 install in another location, pass its path:

```
build.bat C:\path\to\LTTS7
```

### On Windows with MinGW-w64 (32-bit)

The Loquendo DLLs are 32-bit, so use the `i686` MinGW toolchain:

```
mingw32-make
```

GNU `ld` cannot consume the MSVC import library, so the Makefile builds a
MinGW import library (`libloqtts.a`) from `loqtts.def` using `dlltool` with
`--kill-at`. This provides the `@N`-decorated linker symbols that the C
header requires while writing the plain export names into the PE import
table (which is what `LoqTTS7.dll` actually exports).

### On Linux (cross-compile + run under Wine)

```
# from a nix-shell with the 32-bit MinGW cross compiler:
nix-shell -p 'pkgsCross.mingw32.stdenv.cc' --run 'make'
```

The resulting `loqtts.exe` runs in a Wine prefix that has the Loquendo TTS 7
installation (run it from the `bin` directory so the DLLs are found).

### On NixOS with the flake

The repository ships a `flake.nix` that cross-compiles `loqtts.exe` and
provides a development shell with helpers to build and run it under Wine. It
also exposes a `nixosModules.default` so you can install the CLI as part of
your system configuration.

Since the SDK is unfree, set `nixpkgs.config.allowUnfree = true` and build
with `NIXPKGS_ALLOW_UNFREE=1` when needed.

#### NixOS module (recommended)

Import the flake module and point it at your Loquendo TTS 7 installation.
The path must contain the `include/` and `lib/` folders used to build the
CLI.

```nix
# flake.nix (your dotfiles)
{
  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    loqtts.url = "github:<you>/loqtts-cli";
  };

  outputs = { self, nixpkgs, loqtts, ... }: {
    nixosConfigurations.myhost = nixpkgs.lib.nixosSystem {
      system = "x86_64-linux";
      modules = [
        loqtts.nixosModules.default
        ./configuration.nix
      ];
    };
  };
}
```

```nix
# configuration.nix
{
  programs.loqtts = {
    enable = true;
    ltts7 = "/home/nit/Games/loquendo/drive_c/Program Files (x86)/Loquendo/LTTS7";
  };
}
```

Then `sudo nixos-rebuild switch`. This adds `loqtts.exe` and
`LoqTTS7AudioUser.dll` to `environment.systemPackages`, so they end up on
your `PATH` (`/run/current-system/sw/bin`). The binaries are Windows
executables and are meant to be run under Wine (see `loqtts-run` in the dev
shell below).

#### Dev shell

```
nix develop @.
loqtts-build                       # make ROOT=$LTTS7_ROOT
LTTS7_ROOT=/path/to/LTTS7 loqtts-build
loqtts-run --list-voices
loqtts-run -v Carmen -o hello.wav "Hola"
```

`loqtts-build` runs `make ROOT=$LTTS7_ROOT`, so set `LTTS7_ROOT` to your SDK
path (it is no longer baked into the flake). The resulting `loqtts.exe` ends
up in the current directory (or in `result/bin/` if built with `nix build`).

`loqtts-run` copies the freshly built exe into the install's `bin/` (the
Loquendo DLLs resolve by the absolute paths registered in the Wine prefix)
and runs it with `steam-run` + the Wine runner. The relevant environment
variables (`LOQTTS_INSTALL`, `WINEPREFIX`, `WINE64`) can be overridden to
match your setup.

## Usage

```
loqtts [options] [text]
```

| Option | Description |
| ------ | ----------- |
| `-v, --voice <name>` | Voice to use (default: first installed voice) |
| `-l, --language <name>` | Language to use (default: voice's mother tongue) |
| `-s, --style <name>` | Style to use |
| `-r, --rate <wpm>` | Absolute speech rate in words per minute |
| `-p, --pitch <hz>` | Absolute pitch in hertz |
| `-a, --volume <0-100>` | Volume (default 50) |
| `-o, --output <file>` | Write to a file instead of the audio board. `.wav` = WAV file, any other extension = raw PCM. `-` = WAV to stdout |
| `-f, --file <path>` | Read the text to speak from a file |
| `--rate-hz <n>` | Audio sample rate for file output (default 32000) |
| `--mono` | Mono output instead of stereo |
| `--list-voices` | List installed voices and exit |
| `--list-languages` | List installed languages and exit |
| `--version` | Print the Loquendo TTS version and exit |
| `-h, --help` | Show help and exit |

When `[text]` is omitted the program reads the input from `stdin`.

## `-o -` (audio in memory)

`loqtts -o - "text"` writes the synthesized **WAV to stdout** without touching
the disk. This is implemented by the custom audio destination
`LoqTTS7AudioUser.dll` (built from `audiouser.c`): it buffers the PCM that
the engine produces and emits a complete RIFF WAV on stdout at the end of
the stream. The DLL must sit next to `loqtts.exe` (in the SDK `bin/`
directory) so the engine can load it.

```
loqtts -o - "hola" --mono --rate-hz 48000 > audio.wav
```

The Discord bot uses this to synthesize straight into a memory buffer.

## Examples

```
loqtts --list-voices
loqtts -v Carmen -r 180 -p 200 "Esta es una prueba de velocidad"
loqtts -v Carlos -o report.wav -f report.txt
loqtts -v Soledad -o out.pcm --rate-hz 8000 --mono "raw pcm output"
type speech.txt | loqtts -v Jorge
```

## Discord bot (`bot/`)

A Go bot that plays Discord messages in the author's voice channel using
Loquendo voices. It works fully in memory: `loqtts -o -` produces the WAV in
a buffer, which is decoded and encoded to Opus in-process (no ffmpeg, no temp
files).

Since March 2026 Discord requires the **DAVE** end-to-end encryption protocol
for voice. The bot therefore uses
[`disgo`](https://github.com/disgoorg/disgo) with the pure-Go
[`dave-go`](https://github.com/thomas-vilte/dave-go) DAVE implementation
(Go 1.26+), instead of the older `discordgo` (which has no DAVE support and
gets disconnected with close code 4017).

### Commands

* `>tts [opciones] texto` — speak the text. Options: `-v <voz>`, `-l <idioma>`,
  `-s <estilo>`, `-r <palabras/min>`, `-p <hz>`, `-a <0-100>`.
* `>tts stop` — stop the current playback and clear the queue.
* `>ttsmp3 [opciones] texto` — synthesize the text and send the audio as an
  `.mp3` file attached to the message (same options as `>tts`; requires
  `ffmpeg` on the bot's `PATH`).
* `>loqtts --list-voices | --list-languages | --version` — engine info.

### Running

Create a bot in the Discord Developer Portal (token with the **Message
Content** privileged intent enabled), deploy the CLI into the SDK `bin/`, then:

```
nix develop               # go, libopus, steam-run, helpers available
loqtts-build               # make ROOT=$LTTS7_ROOT (set LTTS7_ROOT to your SDK)
loqtts-deploy             # copy loqtts.exe + LoqTTS7AudioUser.dll to LOQTTS_INSTALL/bin
loqtts-bot                # build and run the bot
```

The bot loads its configuration from `bot/.env` automatically (or from a
`.env` in the current directory), so there is **no need to `source` it** —
`source .env` would not even work, because it does not export the variables
to the Go process. Start from `cp bot/.env.example bot/.env` and put your
token in `DISCORD_TOKEN`. Environment variables already set take precedence
over `.env`. You can also run it from the flake package with
`NIXPKGS_ALLOW_UNFREE=1 nix run .#bot`.

Environment variables:

| Variable | Default | Meaning |
| -------- | ------- | ------- |
| `DISCORD_TOKEN` | — (required) | Bot token |
| `BOT_PREFIX` | `>` | Command prefix |
| `TTS_VOICE` | `Carlos` | Default voice |
| `TTS_LANGUAGE` | *empty* | Default language |
| `TTS_RATE` / `TTS_PITCH` / `TTS_VOLUME` | *empty* | Defaults (voice defaults if unset) |
| `LOQTTS_INSTALL` | the LTTS7 path | Install root containing `bin/loqtts.exe` |
| `WINEPREFIX` | `~/Games/loquendo` | Wine prefix |
| `WINE64` | Lutris wine-ge runner | wine64 binary |

You can also run `cd bot && go run ./cmd/pipetest "texto"` to verify the
synthesis→Opus pipeline without Discord (it reads the same `bot/.env` for
the `TTS_*` defaults).

## Notes

* The `LTTS7AudioFile` module writes a RIFF WAV file when the output name
  ends in `.wav`, otherwise a raw PCM stream.
* File output paths are resolved relative to the current directory.
* `build.bat` produces a 32-bit executable that matches the 32-bit
  Loquendo DLLs. Keep the SDK DLLs on the `PATH` (or in the same directory
  as `loqtts.exe`).
