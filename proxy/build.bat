@echo off
rem Build the Engine DJ translation proxy with CMake and run its tests.
rem
rem Default path uses the Visual Studio generator (MSVC x64) and fetches
rem pinned upstream MinHook v1.3.4 when no installed package exists.
rem To consume a vcpkg-installed minhook instead, pass the toolchain file:
rem   build.bat -DCMAKE_TOOLCHAIN_FILE=<vcpkg-root>\scripts\buildsystems\vcpkg.cmake
setlocal
set "HERE=%~dp0"

cmake -S "%HERE%." -B "%HERE%build" -DBUILD_TESTING=ON %*
if errorlevel 1 goto :error

cmake --build "%HERE%build" --config Release
if errorlevel 1 goto :error

ctest --test-dir "%HERE%build" -C Release --output-on-failure
if errorlevel 1 goto :error

echo === Build successful ===
exit /b 0

:error
echo === BUILD FAILED ===
exit /b 1
