"""Synthetic binary fixtures for patcher tests.

Builds minimal-but-real PE images that pefile parses happily: a single
.rdata section whose payload carries NUL-terminated C++ strings and/or an
embedded QML resource blob, mirroring how Engine DJ ships translatabe
content inside the executable.
"""
from __future__ import annotations

import struct
from pathlib import Path

FILE_ALIGNMENT = 0x200
SECTION_ALIGNMENT = 0x1000
RDATA_RVA = SECTION_ALIGNMENT  # 0x1000


def _aligned(value: int, alignment: int) -> int:
    return (value + alignment - 1) // alignment * alignment


def build_pe(path: Path, rdata_payload: bytes) -> dict:
    """Writes a PE32+ image with one .rdata section holding `payload`.

    Returns the layout: rdata file-offset range and payload size.
    """
    dos = bytearray(0x40)
    dos[0:2] = b"MZ"
    struct.pack_into("<I", dos, 0x3C, 0x40)

    coff = struct.pack(
        "<HHIIIHH",
        0x8664,      # Machine = AMD64
        1,           # NumberOfSections
        0,           # TimeDateStamp
        0, 0,        # symbol table
        240,         # SizeOfOptionalHeader (PE32+)
        0x0022,      # Characteristics
    )

    optional = bytearray(240)
    struct.pack_into("<H", optional, 0, 0x20B)              # PE32+ magic
    struct.pack_into("<B", optional, 2, 8)                  # linker major
    struct.pack_into("<I", optional, 16, 0x200)             # SizeOfCode-ish
    struct.pack_into("<I", optional, 24, 0x1000)            # EntryPoint? reserved
    struct.pack_into("<I", optional, 28, 0x2000)            # ImageBase low? PE32+ base @24 is ImageBase (u64) start
    struct.pack_into("<I", optional, 32, SECTION_ALIGNMENT)  # SectionAlignment
    struct.pack_into("<I", optional, 36, FILE_ALIGNMENT)     # FileAlignment
    struct.pack_into("<H", optional, 56, 6)                  # MajorLinkerVersion-ish fields unused by pefile checks here
    struct.pack_into("<I", optional, 60, 0x3000)             # SizeOfImage
    struct.pack_into("<I", optional, 64, 0x400)              # SizeOfHeaders
    struct.pack_into("<I", optional, 68, 0)                  # CheckSum
    struct.pack_into("<H", optional, 70, 0x22)               # Subsystem? GUI=2; value unused
    struct.pack_into("<I", optional, 108, 16)                # NumberOfRvaAndSizes (PE32+ offset 108)

    raw_size = _aligned(len(rdata_payload), FILE_ALIGNMENT)
    virtual_size = _aligned(max(len(rdata_payload), 1), SECTION_ALIGNMENT)
    header_size = 0x400
    section = struct.pack(
        "<8sIIIIIIHHI",
        b".rdata",
        len(rdata_payload),      # VirtualSize
        RDATA_RVA,               # VirtualAddress
        raw_size,                # SizeOfRawData
        header_size,             # PointerToRawData
        0, 0,                    # relocations, linenumbers
        0, 0,
        0x40000040,              # Characteristics: initialized data, read
    )

    headers = bytearray(header_size)
    headers[0 : len(dos)] = dos
    pos = 0x40
    headers[pos : pos + 4] = b"PE\x00\x00"
    pos += 4
    headers[pos : pos + len(coff)] = coff
    pos += len(coff)
    headers[pos : pos + len(optional)] = optional
    pos += len(optional)
    headers[pos : pos + len(section)] = section

    image = bytearray(headers)
    image.extend(b"\x00" * (header_size - len(headers)))
    image.extend(rdata_payload)
    image.extend(b"\x00" * (raw_size - len(rdata_payload)))

    path.write_bytes(bytes(image))
    return {
        "rdata_start": header_size,
        "rdata_size": raw_size,
        "payload_size": len(rdata_payload),
    }


def cpp_string_block(pairs: list[tuple[str, int]]) -> bytes:
    """NUL-terminated strings, each padded with `extra_nuls` trailing NULs."""
    out = bytearray()
    for text, extra_nuls in pairs:
        out += text.encode("utf-8") + b"\x00" * (1 + extra_nuls)
    return bytes(out)


def qml_blob(lines: list[str]) -> bytes:
    """An embedded QML resource the way find_qml_files detects one:
    two NUL bytes, a big-endian u16 length, then the source text.
    Tiny sources are padded with a leading comment so the blob clears
    the detector's 50-byte minimum while staying valid QML."""
    text = "\n".join(lines) + "\n"
    if len(text) < 56:
        text = "// fixture padding " + "#" * (56 - len(text)) + "\n" + text
    source = text.encode("utf-8")
    if len(source) >= 50000:
        raise ValueError("fixture blob too large")
    return b"\x00\x00" + struct.pack(">H", len(source)) + source


def make_source_image(
    path: Path, *, cpp_pairs: list[tuple[str, int]] | None = None,
    qml_lines: list[str] | None = None,
) -> dict:
    payload = bytearray()
    layout = {"qml": None}
    if qml_lines:
        blob = qml_blob(qml_lines)
        layout["qml"] = {
            "offset_hint": len(payload),
            "text": "\n".join(qml_lines) + "\n",
        }
        payload += blob
        payload += b"\x00" * ((8 - len(payload) % 8) % 8 + 2)  # NUL separator
    if cpp_pairs:
        payload += cpp_string_block(cpp_pairs)
        payload += b"\x00" * 64
    if not payload:
        payload = b"\x00" * 128
    info = build_pe(path, bytes(payload))
    layout.update(info)
    return layout


def sha256_file(path: Path) -> str:
    import hashlib
    return hashlib.sha256(Path(path).read_bytes()).hexdigest()
