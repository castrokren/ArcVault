"""Checks for the service-environment registry logic in arcvault_installer.

Run: python installer/windows/test_service_env.py

Covers the parse/format round-trip that decides what SCM actually receives, plus
a read-only probe of the real service key. Installer builds before 2026-07-24
wrote these secrets as REG_SZ values inside an `Environment` SUBKEY, which SCM
never reads -- so the coordinator generated a random JWT secret on every start
and read the credential key off disk. The merge behaviour below is what keeps the
two secrets from overwriting each other.
"""

import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).parent))
from arcvault_installer import ArcVaultInstaller as I  # noqa: E402


def test_parse_and_format():
    assert I.parse_env_entries(["A=1", "B=2"]) == {"A": "1", "B": "2"}

    # A 64-hex secret is the real payload; must survive untouched.
    secret = "a" * 64
    assert I.parse_env_entries([f"ARCVAULT_JWT_SECRET={secret}"]) == {
        "ARCVAULT_JWT_SECRET": secret
    }

    # Values may contain '='; only the first separator counts.
    assert I.parse_env_entries(["PADDED=abc=="]) == {"PADDED": "abc=="}

    # Malformed entries are skipped, not crashed on or turned into empty names.
    assert I.parse_env_entries(["", "NOEQUALS", "=novalue", "OK=1"]) == {"OK": "1"}

    assert I.format_env_entries({"B": "2", "A": "1"}) == ["A=1", "B=2"]

    # Round-trip is lossless -- this is the property SCM depends on.
    original = {"ARCVAULT_CREDENTIAL_KEY": "c" * 64, "ARCVAULT_JWT_SECRET": "j" * 64}
    assert I.parse_env_entries(I.format_env_entries(original)) == original


def test_merge_keeps_both_secrets():
    """Setting one variable must not drop the other.

    Writing the credential key and then the JWT secret with a replace-style write
    would leave only the JWT secret, and credential decryption would 503.
    """
    entries = I.parse_env_entries(I.format_env_entries({"ARCVAULT_CREDENTIAL_KEY": "c" * 64}))
    entries["ARCVAULT_JWT_SECRET"] = "j" * 64
    payload = I.format_env_entries(entries)

    assert payload == [f"ARCVAULT_CREDENTIAL_KEY={'c' * 64}", f"ARCVAULT_JWT_SECRET={'j' * 64}"]
    assert len(I.parse_env_entries(payload)) == 2


def probe_real_service_key():
    """Read-only look at the live service key. Prints names and lengths, never values."""
    inst = I.__new__(I)  # no __init__: avoids building the Tk UI
    env = I._read_service_env(inst, "arcvault-coordinator")
    if not env:
        print("  (no environment entries found for arcvault-coordinator)")
        return
    for name, value in sorted(env.items()):
        print(f"  {name}: {len(value)} chars")


if __name__ == "__main__":
    test_parse_and_format()
    test_merge_keeps_both_secrets()
    print("service-env logic: OK")
    print("live service key, recoverable secrets (values never printed):")
    probe_real_service_key()
