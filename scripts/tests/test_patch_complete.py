"""Regression tests for the structured binary patcher.

Builds in-memory synthetic PE images (see fixtures.py) and drives the
planner/apply/verify/publish pipeline end to end. Every failure scenario
asserts that NO output file is produced.
"""
from __future__ import annotations

import sys
from pathlib import Path

import pytest

sys.path.insert(0, str(Path(__file__).resolve().parent.parent))
sys.path.insert(0, str(Path(__file__).resolve().parent))

import fixtures  # noqa: E402
import patch_complete as pc  # noqa: E402


def make_allowlist(source_path: Path, layout: dict) -> dict:
    return {
        "schema": 1,
        "sources": [
            {
                "sha256": fixtures.sha256_file(source_path),
                "file_version": None,
                "rdata": {
                    "raw_offset": layout["rdata_start"],
                    "raw_size": layout["rdata_size"],
                },
            }
        ],
    }


def write_image(tmp_path: Path, **kwargs):
    source = tmp_path / "Engine DJ.exe.original"
    layout = fixtures.make_source_image(source, **kwargs)
    return source, layout


# ---------------------------- QML planning ----------------------------

def test_tab_inside_string_literal_is_preserved(tmp_path):
    lines = [
        "import QtQuick",
        "Text {",
        "    text: \"Prefs\tX\"",
        "}",
    ]
    source, _ = write_image(tmp_path, qml_lines=lines, cpp_pairs=[])
    data = source.read_bytes()
    translations = {"Prefs\tX": "偏\t设"}

    outcome = pc.plan_qml(data, translations)
    assert len(outcome.replacements) == 1
    patched = bytes(pc.apply_plan(bytearray(data), outcome))
    assert "偏\t设".encode("utf-8") in patched
    # indentation before the property must survive untouched
    assert b"\n    text: " in patched


def test_indentation_tabs_fund_growth(tmp_path):
    lines = [
        "import QtQuick",
        "Item {",
        "\t\ttext: \"Short\"",
        "}",
    ]
    source, _ = write_image(tmp_path, qml_lines=lines, cpp_pairs=[])
    data = source.read_bytes()

    long_cn = "稍长"  # +1 byte over the 5-byte slot; two tabs can fund it
    outcome = pc.plan_qml(data, {"Short": long_cn})
    assert not outcome.failures, outcome.failures

    kinds = {r.kind for r in outcome.replacements}
    assert "qml-indent" in kinds
    assert "qml-string" in kinds
    patched = bytes(pc.apply_plan(bytearray(data), outcome))
    assert long_cn.encode("utf-8") in patched
    assert len(patched) == len(data)


def test_no_translation_means_no_edits(tmp_path):
    lines = ["import QtQuick", "    Item {", "        text: \"OK\"", "    }"]
    source, _ = write_image(tmp_path, qml_lines=lines, cpp_pairs=[])
    data = source.read_bytes()
    outcome = pc.plan_qml(data, {})
    assert outcome.replacements == []
    assert bytes(pc.apply_plan(bytearray(data), outcome)) == data


def test_unfunded_size_growth_is_a_hard_failure(tmp_path):
    lines = [
        "import QtQuick",
        "Text { text: \"Hi\" }",
        "Text { text: \"No indent here at all\" }",
    ]
    source, _ = write_image(tmp_path, qml_lines=lines, cpp_pairs=[])
    data = source.read_bytes()
    big = "远超槽位的中文文本" * 10
    outcome = pc.plan_qml(data, {"Hi": big, "No indent here at all": big})
    assert any(r == "size growth" for r, _ in outcome.failures)


def test_duplicate_occurrences_all_planned(tmp_path):
    lines = [
        "import QtQuick",
        "        Text { text: \"Twin\" }",
        "        Label { text: \"Twin\" }",
    ]
    source, _ = write_image(tmp_path, qml_lines=lines, cpp_pairs=[])
    data = source.read_bytes()
    outcome = pc.plan_qml(data, {"Twin": "双胞胎"})
    assert not outcome.failures, outcome.failures
    string_edits = [r for r in outcome.replacements if r.kind == "qml-string"]
    assert len(string_edits) == 2
    assert any(r.kind == "qml-indent" for r in outcome.replacements)


def test_wrong_expected_bytes_abort_at_apply(tmp_path):
    lines = ["import QtQuick", "Text { text: \"Stable\" }"]

    source, _ = write_image(tmp_path, qml_lines=lines, cpp_pairs=[])
    data = source.read_bytes()
    outcome = pc.plan_qml(data, {"Stable": "稳定"})
    assert outcome.replacements

    tampered = bytearray(data)
    idx = tampered.find(b"Stable")
    tampered[idx] = ord("X")
    with pytest.raises(pc.PatchError, match="context mismatch"):
        pc.apply_plan(tampered, outcome)


# ---------------------------- .rdata planning --------------------------

def test_rdata_shorter_and_growing_patches(tmp_path):
    source, layout = write_image(
        tmp_path,
        cpp_pairs=[("Preferences", 12), ("Tiny", 1)],
        qml_lines=None,
    )
    data = source.read_bytes()
    bounds = (layout["rdata_start"], layout["rdata_size"])

    plan = pc.plan_rdata(
        data,
        {
            "Preferences": "偏好设置",
            "Tiny": "一个明显更长的翻译",
        },
        bounds,
    )
    assert not plan.failures
    patched = bytes(pc.apply_plan(bytearray(data), plan))

    assert "偏好设置".encode("utf-8") in patched
    assert "一个明显更长的翻译".encode("utf-8") in patched
    assert len(patched) == len(data)
    assert b"Preferences\x00" not in patched


def test_rdata_growth_without_padding_fails(tmp_path):
    source, layout = write_image(
        tmp_path,
        cpp_pairs=[("AB", 0)],
        qml_lines=None,
    )
    data = source.read_bytes()
    bounds = (layout["rdata_start"], layout["rdata_size"])
    huge = "远超尾部填充容量的超长中文翻译文本" * 40
    plan = pc.plan_rdata(data, {"AB": huge}, bounds)
    assert any(r == "no NUL padding to grow into" for r, _ in plan.failures)


# ------------------------ allowlist / validation -----------------------

def test_unknown_source_sha_is_rejected(tmp_path):
    source, layout = write_image(tmp_path, cpp_pairs=[("A", 4)])
    allowlist = make_allowlist(source_path=source, layout=layout)
    allowlist["sources"][0]["sha256"] = "0" * 64
    with pytest.raises(pc.PatchError, match="not allowlisted"):
        pc.validate_source(source, allowlist)


def test_allowlisted_rdata_bounds_are_enforced(tmp_path):
    source, layout = write_image(tmp_path, cpp_pairs=[("A", 4)])
    allowlist = make_allowlist(source_path=source, layout=layout)
    allowlist["sources"][0]["rdata"]["raw_offset"] += 0x200
    with pytest.raises(pc.PatchError, match="bounds"):
        pc.locate_rdata(
            source,
            expected={
                "raw_offset": layout["rdata_start"] + 0x200,
                "raw_size": layout["rdata_size"],
            },
        )


# ----------------------- end-to-end publication ------------------------

def test_full_pipeline_publishes_atomically(tmp_path):
    source, layout = write_image(
        tmp_path,
        cpp_pairs=[("Preferences", 16)],
        qml_lines=["import QtQuick", "Text { text: \"About Us\" }"],
    )
    output = tmp_path / "out" / "Engine DJ.patched.exe"
    output.parent.mkdir()
    report = pc.patch_image(
        source,
        output,
        qml_translations={"About Us": "关于"},
        cpp_translations={"Preferences": "偏好设置"},
        allowlist=make_allowlist(source, layout),
    )
    assert report["replacements"] >= 2
    assert output.exists()
    assert not list(output.parent.glob("*.enginetools-tmp"))
    patched = output.read_bytes()
    assert "关于".encode("utf-8") in patched
    assert "偏好设置".encode("utf-8") in patched
    assert len(patched) == source.stat().st_size


def test_any_planning_failure_leaves_no_output(tmp_path):
    source, layout = write_image(
        tmp_path,
        cpp_pairs=[("AB", 0)],
        qml_lines=None,
    )
    output = tmp_path / "never.exe"
    huge = "远超尾部填充容量的超长中文翻译文本" * 40
    with pytest.raises(pc.PatchError):
        pc.patch_image(
            source,
            output,
            qml_translations={},
            cpp_translations={"AB": huge},
            allowlist=make_allowlist(source, layout),
        )
    assert not output.exists()


def test_verification_failure_leaves_no_output(tmp_path):
    source, layout = write_image(tmp_path, cpp_pairs=[("Preferences", 16)])
    output = tmp_path / "never.exe"

    data = source.read_bytes()
    bounds = (layout["rdata_start"], layout["rdata_size"])
    plan = pc.plan_rdata(data, {"Preferences": "偏好设置"}, bounds)
    assert plan.replacements

    sabotaged = pc.PlanOutcome(
        [pc.Replacement(plan.replacements[0].offset,
                        plan.replacements[0].original,
                        b"X" * len(plan.replacements[0].new),
                        plan.replacements[0].kind)],
        [],
    )
    patched = bytearray(pc.apply_plan(bytearray(data), sabotaged))
    off = sabotaged.replacements[0].offset
    patched[off] ^= 0xFF  # corrupt one byte post-apply
    with pytest.raises(pc.PatchError, match="verification failed"):
        pc.verify_patched(
            bytes(patched), sabotaged, len(data), pc.sha256_bytes(data),
            bounds, tmp_path,
        )
    assert not output.exists()
