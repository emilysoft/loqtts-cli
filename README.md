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
provides a development shell with helpers to build and run it under Wine.

The flake takes the (proprietary, non-redistributable) LTTS7 SDK as a `path`
input. Point it at your install with `--override-input`, e.g.:

```
nix build . --override-input loquendo-sdk path:/path/to/LTTS7
nix build . --impure   # with the default path baked in the flake
```

The resulting `loqtts.exe` ends up in `result/bin/`. Then, in a dev shell:

```
nix develop . --impure
loqtts-build            # make ROOT=<flake input SDK>
loqtts-run --list-voices
loqtts-run -v Carmen -o hello.wav "Hola"
```

`loqtts-run` copies the freshly built exe into the install's `bin/` (the
Loquendo DLLs resolve by the absolute paths registered in the Wine prefix)
and runs it with `steam-run` + the Wine runner. The relevant environment
variables (`LTTS7_ROOT`, `LOQTTS_INSTALL`, `WINEPREFIX`, `WINE64`) can be
overridden to match your setup.

Since the SDK is unfree, `--impure` and `NIXPKGS_ALLOW_UNFREE=1` are needed
when building from the flake.

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

* `!tts [opciones] texto` — speak the text. Options: `-v <voz>`, `-l <idioma>`,
  `-s <estilo>`, `-r <palabras/min>`, `-p <hz>`, `-a <0-100>`.
* `!tts stop` — stop the current playback and clear the queue.
* `!loqtts --list-voices | --list-languages | --version` — engine info.

### Running

Create a bot in the Discord Developer Portal (token with the **Message
Content** privileged intent enabled), deploy the CLI into the SDK `bin/`, then:

```
nix develop . --impure   # go, libopus, steam-run, helpers available
loqtts-deploy             # copy loqtts.exe + LoqTTS7AudioUser.dll to LOQTTS_INSTALL/bin
loqtts-bot                # build and run the bot
```

The bot loads its configuration from `bot/.env` automatically (or from a
`.env` in the current directory), so there is **no need to `source` it** —
`source .env` would not even work, because it does not export the variables
to the Go process. Start from `cp bot/.env.example bot/.env` and put your
token in `DISCORD_TOKEN`. Environment variables already set take precedence
over `.env`. You can also run it from the flake package with
`NIXPKGS_ALLOW_UNFREE=1 nix run .#bot --impure`.

Environment variables:

| Variable | Default | Meaning |
| -------- | ------- | ------- |
| `DISCORD_TOKEN` | — (required) | Bot token |
| `BOT_PREFIX` | `!tts` | Command prefix |
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
