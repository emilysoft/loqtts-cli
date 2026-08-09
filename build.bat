@echo off
rem ============================================================
rem  Build loqtts.exe with Microsoft Visual C++ (cl.exe)
rem
rem  Run this from a "Visual Studio Command Prompt" (x86 or x64
rem  tools both work since the import library is 32-bit).
rem
rem  Usage:  build.bat
rem          build.bat C:\path\to\LTTS7   (build against another install)
rem ============================================================
setlocal
set ROOT=%~dp0..
if not "%1"=="" set ROOT=%1

if not exist "%ROOT%\lib\LoqTTS7.lib" (
    echo ERROR: LoqTTS7.lib not found in "%ROOT%\lib"
    exit /b 1
)

cl /nologo /W3 /O2 ^
    /I"%ROOT%\include" ^
    loqtts.c ^
    /link /LIBPATH:"%ROOT%\lib" LoqTTS7.lib ^
    /SUBSYSTEM:CONSOLE /OUT:loqtts.exe

if errorlevel 1 (
    echo Build failed.
    exit /b 1
)

echo.
echo Built loqtts.exe
endlocal
