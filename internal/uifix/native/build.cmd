@echo off
setlocal
if /i "%VSCMD_ARG_TGT_ARCH%"=="x64" goto build
set "ENGINETOOLS_VSWHERE=%ProgramFiles(x86)%\Microsoft Visual Studio\Installer\vswhere.exe"
if not exist "%ENGINETOOLS_VSWHERE%" (
    echo [ERROR] Visual Studio C++ tools not found. Run from an x64 Native Tools Command Prompt.
    exit /b 1
)
for /f "usebackq delims=" %%I in (`"%ENGINETOOLS_VSWHERE%" -latest -products * -requires Microsoft.VisualStudio.Component.VC.Tools.x86.x64 -property installationPath`) do set "ENGINETOOLS_VSROOT=%%I"
if not defined ENGINETOOLS_VSROOT (
    echo [ERROR] Install the Visual Studio Desktop development with C++ workload.
    exit /b 1
)
call "%ENGINETOOLS_VSROOT%\VC\Auxiliary\Build\vcvars64.bat" >nul
if errorlevel 1 exit /b 1
:build
if not exist "%~dp0build" mkdir "%~dp0build"
cd /d "%~dp0build"
if errorlevel 1 (
    echo [ERROR] Cannot enter "%~dp0build".
    exit /b 1
)
cl /nologo /O2 /MT /W4 /c ..\minhook\src\buffer.c ..\minhook\src\hook.c ..\minhook\src\trampoline.c ..\minhook\src\hde\hde64.c
if errorlevel 1 exit /b 1
ml64 /nologo /c ..\src\version_exports.asm
if errorlevel 1 exit /b 1
cl /nologo /std:c++17 /EHsc /O2 /MT /W4 /LD ..\src\interface_cache.cpp ..\src\version_proxy.cpp version_exports.obj buffer.obj hook.obj trampoline.obj hde64.obj bcrypt.lib /link /DEF:..\src\version.def /OUT:version.dll /IMPLIB:version.lib
if errorlevel 1 exit /b 1
copy /y version.dll ..\..\assets\version.bin >nul
if errorlevel 1 exit /b 1
