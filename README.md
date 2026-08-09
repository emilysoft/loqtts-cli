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
| `-o, --output <file>` | Write to a file instead of the audio board. `.wav` = WAV file, any other extension = raw PCM |
| `-f, --file <path>` | Read the text to speak from a file |
| `--rate-hz <n>` | Audio sample rate for file output (default 32000) |
| `--mono` | Mono output instead of stereo |
| `--list-voices` | List installed voices and exit |
| `--list-languages` | List installed languages and exit |
| `--version` | Print the Loquendo TTS version and exit |
| `-h, --help` | Show help and exit |

When `[text]` is omitted the program reads the input from `stdin`.

## Examples

```
loqtts --list-voices
loqtts -v Carmen -r 180 -p 200 "Esta es una prueba de velocidad"
loqtts -v Carlos -o report.wav -f report.txt
loqtts -v Soledad -o out.pcm --rate-hz 8000 --mono "raw pcm output"
type speech.txt | loqtts -v Jorge
```

## Notes

* The `LTTS7AudioFile` module writes a RIFF WAV file when the output name
  ends in `.wav`, otherwise a raw PCM stream.
* File output paths are resolved relative to the current directory.
* `build.bat` produces a 32-bit executable that matches the 32-bit
  Loquendo DLLs. Keep the SDK DLLs on the `PATH` (or in the same directory
  as `loqtts.exe`).
