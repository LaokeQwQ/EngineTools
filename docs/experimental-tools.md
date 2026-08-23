# Experimental Tools

This repository ships one released product: the EngineTools Wails desktop
application (Go + Vue). Alongside it, this working copy carries several
experimental artifacts used while investigating Engine DJ translation and
binary patching. They are **not** part of the release build.

| Path | Purpose | Status |
| --- | --- | --- |
| `proxy/` | `version.dll` proxy that hooks Engine DJ to apply runtime string translations. Currently a handwritten hook engine plus unbounded string lookups; being replaced with upstream MinHook under tasks 3–4 of the hardening plan. | Experimental |
| `scripts/*.py` | QML string extraction, translation workbook tooling, and PE patchers for `Engine DJ.exe`. Being restructured to parse PE structure and fail closed under task 6. | Experimental |
| `deploy*.ps1`, `restore_original.ps1` | Ad-hoc deployment/restore scripts for the patched binary. Being replaced by a journaled PowerShell transaction under task 5. | Experimental |
| `Engine_DJ_*.xlsx` | Translation workbooks (string tables feeding the proxy/patcher). | Experimental data |

## Ignored vs. tracked

- Local backups (`backups/`), logs (`*.log`), debug extracts
  (`debug_*.txt`, `translation_check.txt`), deploy results, Python caches,
  compiled proxy binaries (`*.o`, `*.dll`) and generated patched EXEs are
  ignored via `.gitignore`; they stay on disk but never get staged.
- `.claude/settings.local.json` is ignored; it holds local assistant settings.
- XLSX workbooks are deliberately **not** blanket-ignored: they are the source
  data for translations and may be committed intentionally once reviewed.
- Everything else stays untracked until its hardening task lands; nothing is
  deleted from the working tree by hygiene tasks.
