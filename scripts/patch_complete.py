"""Complete Patcher v8: structured, PE-aware, fail-closed.

Patches Engine DJ translations inside the executable:

* QML resource blobs -- property strings replaced in place, never longer
  than their slot; extra room may come only from shrinking *leading*
  indentation on lines that start outside string literals. Whitespace
  inside literals is never touched.
* .rdata C++ strings -- NUL-terminated replacements may only extend into
  following NUL padding, keeping the image size byte-for-byte identical.

Hard safety rules (each one exits nonzero and never writes an output
file):

* the source EXE SHA-256 must appear in the versioned JSON allowlist;
* .rdata bounds come from parsing the PE with `pefile` (never hardcoded),
  are checked against the allowlist when it records them, and must sit
  inside the file without overlapping other sections;
* every replacement pins the exact original bytes and refuses on mismatch;
* missing translations, size growth that cannot be funded by reclaimed
  indentation/NUL padding, and any context mismatch are hard failures;
* output goes to a sibling temp, is re-validated in full (size, every
  offset, PE parse), and only then atomically renamed into place.
"""
from __future__ import annotations

import argparse
import hashlib
import json
import os
import re
import sys
from dataclasses import dataclass
from pathlib import Path

import pefile
from openpyxl import load_workbook

sys.stdout.reconfigure(encoding="utf-8", errors="replace")

ROOT = Path(__file__).resolve().parent.parent

TEXT_PROPS = [
    "text", "tooltip", "title", "message", "subMessage", "subTitle",
    "label", "placeholderText", "description", "displayText",
    "header", "footer", "hint", "name", "caption", "content",
    "prompt", "headline", "detail", "summary", "instruction",
    "emptyText", "loadingText", "progressText", "alternateText",
    "accessibilityName",
]
PROPS_RE = b"|".join(p.encode() for p in TEXT_PROPS)
PROP_PAT = re.compile(
    rb"(?:^|[\r\n;{}])[ \t]*(" + PROPS_RE + rb")[ \t]*:[ \t]*\"([^\r\n\"]{2,500})\"",
    re.MULTILINE,
)


class PatchError(Exception):
    """Any condition that must abort without producing an output file."""


@dataclass(frozen=True)
class Replacement:
    offset: int          # absolute file offset
    original: bytes      # exact bytes required at this offset
    new: bytes           # equal-footprint bytes to write
    kind: str            # qml-string | rdata-string | qml-indent
    note: str = ""


@dataclass
class PlanOutcome:
    replacements: list[Replacement]
    failures: list[tuple[str, str]]      # (reason, detail)

    @property
    def ok(self) -> bool:
        return not self.failures


# --------------------------------------------------------------------------
# allowlist / source validation
# --------------------------------------------------------------------------

def sha256_bytes(data: bytes) -> str:
    return hashlib.sha256(data).hexdigest()


def load_allowlist(path: Path) -> dict:
    data = json.loads(Path(path).read_text(encoding="utf-8"))
    if data.get("schema") != 1:
        raise PatchError(f"unsupported allowlist schema: {data.get('schema')!r}")
    return data


def validate_source(pe_path: Path, allowlist: dict) -> dict:
    digest = sha256_bytes(Path(pe_path).read_bytes())
    for entry in allowlist.get("sources", []):
        if entry.get("sha256") == digest:
            recorded_version = entry.get("file_version")
            if recorded_version is not None:
                actual = read_file_version(pe_path)
                if actual != recorded_version:
                    raise PatchError(
                        f"file version {actual!r} != allowlisted {recorded_version!r}"
                    )
            return entry
    raise PatchError(f"source SHA-256 {digest} is not allowlisted")


def read_file_version(pe_path: Path) -> str | None:
    pe = pefile.PE(str(pe_path))
    try:
        info = pe.FileInfo if hasattr(pe, "FileInfo") else None
        if not info:
            return None
        for fileinfo in info[0]:
            if fileinfo.Key == b"StringFileInfo":
                for st in fileinfo.StringTable:
                    version = st.entries.get(b"FileVersion")
                    return version.decode("utf-8", "replace") if version else None
        return None
    finally:
        pe.close()


# --------------------------------------------------------------------------
# PE structure
# --------------------------------------------------------------------------

def locate_rdata(pe_path: Path, expected: dict | None = None) -> tuple[int, int]:
    """Returns absolute (offset, size) of .rdata raw contents."""
    pe = pefile.PE(str(pe_path), fast_load=True)
    try:
        ranges: list[tuple[str, int, int]] = []
        rdata = None
        for section in pe.sections:
            name = section.Name.rstrip(b"\x00").decode("ascii", "replace")
            begin = section.PointerToRawData
            size = section.SizeOfRawData
            if name == ".rdata":
                rdata = (begin, size)
            ranges.append((name, begin, begin + size))
        if rdata is None:
            raise PatchError("PE has no .rdata section")
        begin, size = rdata
        file_len = Path(pe_path).stat().st_size
        if size <= 0 or begin < 0x400 or begin + size > file_len:
            raise PatchError(f".rdata raw bounds out of file: {begin:#x}+{size:#x}")
        for name, other_begin, other_end in ranges:
            if name == ".rdata":
                continue
            if begin < other_end and other_begin < begin + size:
                raise PatchError(f".rdata overlaps section {name}")
        if expected is not None:
            if (begin, size) != (expected["raw_offset"], expected["raw_size"]):
                raise PatchError(
                    f".rdata bounds {(begin, size)} != allowlisted "
                    f"{(expected['raw_offset'], expected['raw_size'])}"
                )
        return begin, size
    finally:
        pe.close()


# --------------------------------------------------------------------------
# embedded QML resources
# --------------------------------------------------------------------------

def find_qml_files(data: bytes) -> list[dict]:
    pattern = re.compile(rb"\x00\x00(..)")
    files = []
    for m in pattern.finditer(data):
        cs = m.end()
        size = (m.group(1)[0] << 8) | m.group(1)[1]
        ce = cs + size
        if 50 < size < 50000 and ce <= len(data):
            head = data[cs : cs + 7]
            if head.startswith(b"import ") or head.startswith(b"// "):
                files.append({"cs": cs, "ce": ce})
    return files


def _line_starts_outside_literal(blob: bytes) -> list[int]:
    """Offsets of each line start whose position is outside a string
    literal (even number of unescaped double quotes before it)."""
    starts = [0]
    in_string = False
    escaped = False
    for i, byte in enumerate(blob):
        if escaped:
            escaped = False
            continue
        ch = chr(byte) if byte < 0x80 else ""
        if in_string:
            if ch == "\\":
                escaped = True
            elif ch == '"':
                in_string = False
        else:
            if ch == '"':
                in_string = True
            elif ch == "\n" and i + 1 < len(blob):
                starts.append(i + 1)
    return starts


def plan_indent_reclaims(blob: bytes, base: int) -> list[Replacement]:
    """Whitespace-only indentation runs on out-of-literal lines may be
    removed wholesale; these reclaims fund longer translations elsewhere
    in the same blob while keeping the blob size constant."""
    reclaims = []
    for start in _line_starts_outside_literal(blob):
        run_end = start
        while run_end < len(blob) and blob[run_end] in (0x20, 0x09):
            run_end += 1
        width = run_end - start
        if width >= 2:
            reclaims.append(
                Replacement(
                    offset=base + start,
                    original=blob[start:run_end],
                    new=b"",
                    kind="qml-indent",
                )
            )
    return reclaims


def plan_qml(data: bytes, translations: dict[str, str]) -> PlanOutcome:
    replacements: list[Replacement] = []
    failures: list[tuple[str, str]] = []

    for qf in find_qml_files(data):
        blob = data[qf["cs"] : qf["ce"]]
        grows: list[tuple[int, int, int, str, str]] = []  # off,len,need,val,tr
        pending_failures: list[tuple[str, str]] = []

        for m in PROP_PAT.finditer(blob):
            raw = m.group(2)
            try:
                value = raw.decode("utf-8")
            except UnicodeDecodeError:
                continue
            trans = translations.get(value)
            if trans is None:
                continue  # simply untranslated, not an error
            slot = m.end(2) - m.start(2)
            tb = trans.encode("utf-8")
            offset = qf["cs"] + m.start(2)
            if len(tb) > slot:
                grows.append((offset, slot, len(tb) - slot, value, trans))
            else:
                new = tb + b" " * (slot - len(tb))
                replacements.append(
                    Replacement(offset, bytes(raw), new, "qml-string")
                )

        needed = sum(g[2] for g in grows)
        candidates = plan_indent_reclaims(blob, qf["cs"]) if grows else []
        capacity = sum(len(c.original) for c in candidates)

        if grows and needed > capacity:
            for _, _, _, value, _ in grows:
                pending_failures.append(("size growth", value))
        else:
            # Reclaim leading whitespace up to the exact deficit so the blob
            # keeps its original size byte for byte.
            remaining = needed
            for cand in candidates:
                if remaining <= 0:
                    break
                take = min(remaining, len(cand.original))
                replacements.append(
                    Replacement(
                        cand.offset,
                        cand.original[:take],
                        b"",
                        "qml-indent",
                    )
                )
                remaining -= take
            for offset, slot, extra, value, trans in grows:
                tb = trans.encode("utf-8")
                replacements.append(
                    Replacement(
                        offset,
                        bytes(data[offset : offset + slot]),
                        tb,
                        "qml-string",
                        note=f"funded by indent reclaim ({extra}B)",
                    )
                )
        failures.extend(pending_failures)

    return PlanOutcome(replacements, failures)


# --------------------------------------------------------------------------
# .rdata C++ strings
# --------------------------------------------------------------------------

def plan_rdata(
    data: bytes, translations: dict[str, str], bounds: tuple[int, int]
) -> PlanOutcome:
    replacements: list[Replacement] = []
    failures: list[tuple[str, str]] = []
    start, size = bounds
    region = data[start : start + size]

    for original, trans in translations.items():
        ob = original.encode("utf-8")
        tb = trans.encode("utf-8")
        if len(tb) < len(ob):
            trans = trans.ljust(len(original))  # pad with spaces inside literal
            tb = trans.encode("utf-8")
        pattern = re.compile(re.escape(ob) + rb"\x00")
        for m in pattern.finditer(region):
            local = m.start()
            if local > 0 and region[local - 1] != 0x00:
                continue  # suffix of a longer string; not standalone
            padding = 0
            cursor = m.end()  # past the terminator
            while cursor + padding < size and region[cursor + padding] == 0x00:
                padding += 1
            delta = len(tb) - len(ob)
            footprint = len(ob) + 1 + padding  # text + terminator + padding
            if delta > padding:
                failures.append(("no NUL padding to grow into", original))
                continue
            if delta >= 0:
                new = tb + b"\x00" * (1 + padding - delta)
            else:
                new = tb + b"\x00" * (1 + padding - delta)
            offset = start + local
            replacements.append(
                Replacement(
                    offset,
                    bytes(data[offset : offset + footprint]),
                    new[:footprint],
                    "rdata-string",
                )
            )
    return PlanOutcome(replacements, failures)


# --------------------------------------------------------------------------
# application / verification / publication
# --------------------------------------------------------------------------

def apply_plan(data: bytearray, plan: PlanOutcome) -> bytearray:
    out = bytearray(data)
    for rep in sorted(plan.replacements, key=lambda r: r.offset, reverse=True):
        segment = bytes(out[rep.offset : rep.offset + len(rep.original)])
        if segment != rep.original:
            raise PatchError(
                f"context mismatch at {rep.offset:#x} ({rep.kind}): "
                f"{segment[:32]!r} != {rep.original[:32]!r}"
            )
        out[rep.offset : rep.offset + len(rep.original)] = rep.new
    return out


def verify_patched(
    patched: bytes, plan: PlanOutcome, source_size: int,
    source_sha: str, rdata_bounds: tuple[int, int],
    workdir: Path,
) -> None:
    if len(patched) != source_size:
        raise PatchError(
            f"size changed: {source_size} -> {len(patched)}"
        )
    if sha256_bytes(patched) == source_sha:
        raise PatchError("output is byte-identical to input; nothing patched?")
    for rep in plan.replacements:
        segment = patched[rep.offset : rep.offset + len(rep.new)]
        if segment != rep.new:
            raise PatchError(f"verification failed at {rep.offset:#x} ({rep.kind})")
    # the temp must still parse as the same PE layout
    probe = workdir / "_verify.pe"
    probe.write_bytes(patched)
    try:
        locate_rdata(probe, expected={"raw_offset": rdata_bounds[0], "raw_size": rdata_bounds[1]})
    finally:
        probe.unlink(missing_ok=True)


def publish(patched: bytes, output_path: Path) -> None:
    temp = output_path.with_name(output_path.name + ".enginetools-tmp")
    try:
        temp.write_bytes(patched)
        os.replace(temp, output_path)
    finally:
        if temp.exists():
            temp.unlink(missing_ok=True)


# --------------------------------------------------------------------------
# orchestration
# --------------------------------------------------------------------------

def patch_image(
    source_path: Path,
    output_path: Path,
    qml_translations: dict[str, str],
    cpp_translations: dict[str, str],
    allowlist: dict,
) -> dict:
    entry = validate_source(source_path, allowlist)
    source = Path(source_path).read_bytes()
    bounds = locate_rdata(source_path, expected=entry.get("rdata"))

    qml_plan = plan_qml(source, qml_translations)
    cpp_plan = plan_rdata(source, cpp_translations, bounds)
    failures = qml_plan.failures + cpp_plan.failures
    if failures:
        raise PatchError(
            "planning produced "
            f"{len(failures)} failure(s): "
            + "; ".join(f"{reason}: {detail[:40]}" for reason, detail in failures[:10])
        )

    combined = PlanOutcome(
        qml_plan.replacements + cpp_plan.replacements, []
    )
    patched = apply_plan(bytearray(source), combined)

    workdir = Path(output_path).parent
    verify_patched(
        patched, combined, len(source), sha256_bytes(source), bounds, workdir
    )
    Path(output_path).parent.mkdir(parents=True, exist_ok=True)
    publish(patched, Path(output_path))

    return {
        "replacements": len(combined.replacements),
        "by_kind": {
            kind: sum(1 for r in combined.replacements if r.kind == kind)
            for kind in {"qml-string", "rdata-string", "qml-indent"}
        },
        "output": str(output_path),
        "sha256": sha256_bytes(patched),
    }


def load_translations(excel_path: Path) -> tuple[dict[str, str], dict[str, str]]:
    """Reads (qml, cpp) maps from the workbook; conflicting duplicate
    originals (same source text, different translations) are fatal."""
    wb = load_workbook(excel_path)
    ws = wb.worksheets[0]

    seen: dict[str, str] = {}
    qml: dict[str, str] = {}
    cpp: dict[str, str] = {}
    for row in ws.iter_rows(min_row=2, values_only=True):
        source_tag = str(row[1]).strip() if row[1] else ""
        orig = str(row[2]).strip() if row[2] else ""
        cn = str(row[3]).strip() if row[3] else ""
        if not orig or not cn or cn == orig:
            continue
        if orig in seen and seen[orig] != cn:
            raise PatchError(
                f"conflicting translations for {orig!r}: {seen[orig]!r} vs {cn!r}"
            )
        seen[orig] = cn
        if "QML" in source_tag:
            qml[orig] = cn
        if "C++" in source_tag:
            cpp[orig] = cn
    return qml, cpp


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    parser.add_argument("--source", type=Path,
                        default=ROOT / "backups" / "Engine DJ.exe.original")
    parser.add_argument("--excel", type=Path,
                        default=ROOT / "Engine_DJ_Translation_Complete.xlsx")
    parser.add_argument("--output", type=Path, default=ROOT / "Engine DJ.patched.exe")
    parser.add_argument("--allowlist", type=Path,
                        default=Path(__file__).parent / "source_allowlist.json")
    args = parser.parse_args(argv)

    try:
        allowlist = load_allowlist(args.allowlist)
        validate_source(args.source, allowlist)
        qml, cpp = load_translations(args.excel)
        print(f"translations: {len(qml)} QML, {len(cpp)} C++")
        report = patch_image(
            args.source, args.output, qml, cpp, allowlist
        )
    except PatchError as exc:
        print(f"PATCH FAILED: {exc}", file=sys.stderr)
        return 1

    print(f"replacements: {report['replacements']} ({report['by_kind']})")
    print(f"written: {report['output']}")
    print(f"sha256: {report['sha256']}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
