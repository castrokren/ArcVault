from __future__ import annotations

import shlex
from typing import List, Tuple

from .config import BackupConfig


def build_robocopy_commands(cfg: BackupConfig) -> List[List[str]]:
    """Build one robocopy command per source.

    Returns a list of argv lists, e.g. [["robocopy", src, dest, ...], ...]
    """
    base_flags: List[str] = []
    if cfg.options.include_empty_dirs:
        base_flags.append("/E")
    if cfg.options.verbose:
        base_flags.append("/V")
    if cfg.options.report_extras:
        base_flags.append("/X")
    base_flags.append(f"/R:{cfg.options.retries}")
    base_flags.append(f"/W:{cfg.options.wait_seconds}")
    base_flags.append(f"/MT:{cfg.options.threads}")
    if cfg.options.remove_system_hidden:
        base_flags.append("/A-:SH")
    if cfg.options.copy_symlinks:
        base_flags.append("/SL")

    commands: List[List[str]] = []
    for src in cfg.sources:
        argv = [
            "robocopy",
            src,
            cfg.destination,
            *base_flags,
        ]
        commands.append(argv)
    return commands


def classify_exit_code(code: int) -> Tuple[str, str]:
    """Map robocopy exit codes to (state, reason).

    Based on Microsoft docs: 0 and 1 are success; >=8 indicates failure.
    """
    if code in (0, 1):
        return ("succeeded", f"robocopy exit={code}")
    if 2 <= code <= 7:
        return ("succeeded", f"robocopy exit={code} (extra/ignored differences)")
    return ("failed", f"robocopy exit={code}")


