from __future__ import annotations

import subprocess
from typing import Callable, Iterable, Iterator, List, Tuple, Optional

from .config import BackupConfig
from .robocopy import build_robocopy_commands, classify_exit_code


LogSink = Callable[[str], None]


def _unc_share_root(path: str) -> Optional[str]:
    if path.startswith("\\\\"):
        parts = path.strip("\\").split("\\")
        if len(parts) >= 2:
            return f"\\\\{parts[0]}\\{parts[1]}"
    return None


def _net_use_connect(share: str, username: Optional[str], password: Optional[str], domain: Optional[str], emit: LogSink) -> None:
    if not share:
        return
    user = None
    if username:
        user = f"{domain}\\{username}" if domain else username
    args = ["net", "use", share]
    if password:
        args.append(password)
    if user:
        args.append(f"/user:{user}")
    emit(f"NET USE connect {share}")
    subprocess.run(args, capture_output=True, text=True)


def _net_use_disconnect(share: str, emit: LogSink) -> None:
    if not share:
        return
    args = ["net", "use", share, "/delete", "/y"]
    emit(f"NET USE disconnect {share}")
    subprocess.run(args, capture_output=True, text=True)


def run_backup(cfg: BackupConfig, emit: LogSink) -> Tuple[str, str]:
    """Execute robocopy for each source and emit logs in real-time.

    Returns (final_state, reason).
    """
    commands = build_robocopy_commands(cfg)
    src_share = _unc_share_root(cfg.sources[0]) if cfg.sources else None
    dst_share = _unc_share_root(cfg.destination)
    try:
        if cfg.source_credentials and src_share:
            _net_use_connect(src_share, cfg.source_credentials.username, cfg.source_credentials.password, cfg.source_credentials.domain, emit)
        if cfg.destination_credentials and dst_share:
            _net_use_connect(dst_share, cfg.destination_credentials.username, cfg.destination_credentials.password, cfg.destination_credentials.domain, emit)

        for idx, argv in enumerate(commands, start=1):
            emit(f"Executing robocopy {idx}/{len(commands)}: {' '.join(argv)}")
            proc = subprocess.Popen(argv, stdout=subprocess.PIPE, stderr=subprocess.STDOUT, text=True)
            assert proc.stdout is not None
            for line in iter(proc.stdout.readline, ''):
                if not line:
                    break
                emit(line.rstrip('\n'))
            rc = proc.wait()
            state, reason = classify_exit_code(rc)
            emit(f"robocopy finished with code {rc} -> {state}")
            if state != "succeeded":
                return state, reason
        return "succeeded", "all sources processed"
    finally:
        if src_share:
            _net_use_disconnect(src_share, emit)
        if dst_share:
            _net_use_disconnect(dst_share, emit)


