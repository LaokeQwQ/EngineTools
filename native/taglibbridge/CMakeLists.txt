cmake_minimum_required(VERSION 3.21)
project(enginetools_tagbridge LANGUAGES CXX)

set(CMAKE_CXX_STANDARD 17)
set(CMAKE_CXX_STANDARD_REQUIRED ON)
set(CMAKE_CXX_EXTENSIONS OFF)
set(CMAKE_WINDOWS_EXPORT_ALL_SYMBOLS ON)

# TagLib 2.0.2 must already be built and importable: either an installed
# package (find_package) or the sibling directory produced by
# build/windows/build-taglib.ps1.
set(TAGLIB_ROOT "" CACHE PATH "Prebuilt TagLib 2.0.2 install prefix")
if(TAGLIB_ROOT)
  # When a prefix is supplied, never fall back to a machine-wide package:
  # release builds must link the pinned TagLib tree passed by the build script.
  find_package(TagLib 2.0.2 EXACT CONFIG QUIET
    PATHS "${TAGLIB_ROOT}" NO_DEFAULT_PATH)
else()
  find_package(TagLib 2.0.2 CONFIG QUIET)
endif()
if(NOT TARGET TagLib::tag AND NOT TARGET TagLib::TagLib)
  if(NOT TAGLIB_ROOT)
    message(FATAL_ERROR "TagLib not found; set TAGLIB_ROOT or run build-taglib.ps1")
  endif()
  if(NOT EXISTS "${TAGLIB_ROOT}/bin/tag.dll" OR
     NOT EXISTS "${TAGLIB_ROOT}/lib/tag.lib" OR
     NOT EXISTS "${TAGLIB_ROOT}/include/taglib")
    message(FATAL_ERROR "TAGLIB_ROOT does not contain a usable TagLib install: ${TAGLIB_ROOT}")
  endif()
  add_library(taglib SHARED IMPORTED)
  set_target_properties(taglib PROPERTIES
    IMPORTED_LOCATION "${TAGLIB_ROOT}/bin/tag.dll"
    IMPORTED_IMPLIB   "${TAGLIB_ROOT}/lib/tag.lib"
    INTERFACE_INCLUDE_DIRECTORIES "${TAGLIB_ROOT}/include")
  set(TAGLIB_TARGET taglib)
else()
  # The installed TagLib package exports TagLib::tag (and an alias
  # TagLib::TagLib).  Keep the target selected here so both package and
  # TAGLIB_ROOT builds link the same way.
  if(TARGET TagLib::tag)
    set(TAGLIB_TARGET TagLib::tag)
  else()
    set(TAGLIB_TARGET TagLib::TagLib)
  endif()
endif()

add_library(enginetools_tag SHARED src/enginetools_tag.cpp)
target_include_directories(enginetools_tag PUBLIC include)
target_compile_definitions(enginetools_tag PRIVATE WIN32_LEAN_AND_MEAN NOMINMAX UNICODE _UNICODE)
if(MSVC)
  target_compile_options(enginetools_tag PRIVATE /W4 /EHsc /permissive- /utf-8)
endif()
target_link_libraries(enginetools_tag PRIVATE ${TAGLIB_TARGET})

if(BUILD_TESTING)
  enable_testing()
  add_executable(bridge_test tests/bridge_test.cpp)
  target_link_libraries(bridge_test PRIVATE enginetools_tag)
  if(MSVC)
    target_compile_options(bridge_test PRIVATE /W4 /EHsc /permissive- /utf-8)
  endif()
  # Copy a shared TagLib runtime beside the test binary so CTest exercises the
  # real DLL boundary.  Static package targets do not have a runtime file.
  get_target_property(TAGLIB_TYPE "${TAGLIB_TARGET}" TYPE)
  if(WIN32 AND (TAGLIB_TYPE STREQUAL "SHARED_LIBRARY" OR
                TAGLIB_TYPE STREQUAL "MODULE_LIBRARY"))
    add_custom_command(TARGET bridge_test POST_BUILD
      COMMAND ${CMAKE_COMMAND} -E copy_if_different
        "$<TARGET_FILE:${TAGLIB_TARGET}>" "$<TARGET_FILE_DIR:bridge_test>")
  endif()
  add_test(NAME bridge COMMAND $<TARGET_FILE:bridge_test>)
endif()
