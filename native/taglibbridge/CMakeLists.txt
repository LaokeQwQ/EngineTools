cmake_minimum_required(VERSION 3.21)
project(enginetools_tagbridge LANGUAGES CXX)

set(CMAKE_CXX_STANDARD 17)
set(CMAKE_CXX_STANDARD_REQUIRED ON)
set(CMAKE_CXX_EXTENSIONS OFF)
set(CMAKE_WINDOWS_EXPORT_ALL_SYMBOLS ON)

# TagLib 2.0.2 must already be built and importable: either an installed
# package (find_package) or the sibling directory produced by
# build/windows/build-taglib.ps1.
find_package(taglib CONFIG QUIET)
if(NOT taglib_FOUND)
  set(TAGLIB_ROOT "" CACHE PATH "Prebuilt TagLib 2.0.2 install prefix")
  if(NOT TAGLIB_ROOT)
    message(FATAL_ERROR "TagLib not found; set TAGLIB_ROOT or run build-taglib.ps1")
  endif()
  add_library(taglib SHARED IMPORTED)
  set_target_properties(taglib PROPERTIES
    IMPORTED_LOCATION "${TAGLIB_ROOT}/bin/tag.dll"
    IMPORTED_IMPLIB   "${TAGLIB_ROOT}/lib/tag.lib"
    INTERFACE_INCLUDE_DIRECTORIES "${TAGLIB_ROOT}/include")
  set(MINHOOK_TARGET taglib)
else()
  set(MINHOOK_TARGET TagLib::tag)
endif()

add_library(enginetools_tag SHARED src/enginetools_tag.cpp)
target_include_directories(enginetools_tag PUBLIC include)
target_compile_definitions(enginetools_tag PRIVATE WIN32_LEAN_AND_MEAN NOMINMAX UNICODE _UNICODE)
if(MSVC)
  target_compile_options(enginetools_tag PRIVATE /W4 /EHsc /permissive- /utf-8)
endif()
target_link_libraries(enginetools_tag PRIVATE taglib)

if(BUILD_TESTING)
  enable_testing()
  add_executable(bridge_test tests/bridge_test.cpp)
  target_link_libraries(bridge_test PRIVATE enginetools_tag)
  add_test(NAME bridge COMMAND bridge_test)
endif()
