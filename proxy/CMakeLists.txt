cmake_minimum_required(VERSION 3.21)
project(enginetools_proxy VERSION 2.0.0 LANGUAGES CXX)

set(CMAKE_CXX_STANDARD 17)
set(CMAKE_CXX_STANDARD_REQUIRED ON)
set(CMAKE_CXX_EXTENSIONS OFF)

option(BUILD_TESTING "Build test targets" ON)
include(CTest)
include(FetchContent)

# Upstream MinHook, pinned to release v1.3.4. Prefer an installed config
# package (vcpkg manifest in this directory + toolchain file); otherwise
# fetch exactly this tag so builds are reproducible without external state.
find_package(minhook CONFIG QUIET)
if(NOT minhook_FOUND)
  FetchContent_Declare(minhook
    GIT_REPOSITORY https://github.com/TsudaKageyu/minhook.git
    GIT_TAG        v1.3.4)
  FetchContent_MakeAvailable(minhook)
endif()

if(TARGET minhook::libminhook)
  set(MINHOOK_TARGET minhook::libminhook)
elseif(TARGET MinHook::libminhook)
  set(MINHOOK_TARGET MinHook::libminhook)
elseif(TARGET libminhook)
  set(MINHOOK_TARGET libminhook)
else()
  message(FATAL_ERROR "MinHook target not found after find_package/FetchContent")
endif()

add_library(proxy SHARED main.cpp version.def)
target_compile_definitions(proxy PRIVATE WIN32_LEAN_AND_MEAN NOMINMAX)
if(MSVC)
  target_compile_options(proxy PRIVATE /W4 /EHsc /permissive-)
else()
  target_compile_options(proxy PRIVATE -Wall -Wextra)
endif()
target_link_libraries(proxy PRIVATE ${MINHOOK_TARGET})

if(BUILD_TESTING)
  # lookup_test is portable and also runs on non-Windows hosts.
  add_executable(lookup_test tests/lookup_test.cpp)
  if(MSVC)
    target_compile_options(lookup_test PRIVATE /W4 /EHsc)
  else()
    target_compile_options(lookup_test PRIVATE -Wall -Wextra)
  endif()
  add_test(NAME lookup COMMAND lookup_test)

  if(WIN32)
    add_executable(hook_smoke_test tests/hook_smoke_test.cpp)
    target_link_libraries(hook_smoke_test PRIVATE ${MINHOOK_TARGET})
    if(MSVC)
      target_compile_options(hook_smoke_test PRIVATE /W4 /EHsc)
    endif()
    add_test(NAME hook_smoke COMMAND hook_smoke_test)
  endif()
endif()
