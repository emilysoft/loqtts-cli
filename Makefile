# ============================================================
#  Build loqtts.exe with MinGW-w64 (32-bit target)
#
#  The Loquendo TTS 7 DLLs and import library are 32-bit, so
#  you must use the i686 (32-bit) MinGW compiler.
#
#  The MSVC import library (LoqTTS7.lib) is not usable by the
#  GNU linker, so the import library is generated from
#  loqtts.def with dlltool (part of the MinGW binutils).
#
#  Usage:
#     make                 - build loqtts.exe
#     make ROOT=<ltss7>    - build against an LTTS7 install elsewhere
#     make clean
#
#  ROOT points at the Loquendo TTS 7 install directory (which must
#  contain include/ and lib/).  It defaults to "..", so this folder
#  can simply be placed inside the LTTS7 install (e.g. LTTS7/cli/).
# ============================================================

CC      = i686-w64-mingw32-gcc
DLLTOOL = i686-w64-mingw32-dlltool
ROOT    ?= ..
CFLAGS  = -O2 -Wall -I"$(ROOT)/include"
LDFLAGS = -static-libgcc

all: loqtts.exe

libloqtts.a: loqtts.def
	$(DLLTOOL) --kill-at --dllname LoqTTS7.dll --input-def loqtts.def --output-lib $@

loqtts.exe: loqtts.c libloqtts.a
	$(CC) $(CFLAGS) loqtts.c libloqtts.a $(LDFLAGS) -o $@

clean:
	rm -f loqtts.exe libloqtts.a
