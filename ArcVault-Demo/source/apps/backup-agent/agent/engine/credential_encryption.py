"""
Secure credential encryption utilities using Fernet symmetric encryption.

Passwords are encrypted at rest and decrypted only when needed for use.
Encryption key is derived from environment variable or system properties.
"""
import os
import base64
from typing import Optional
from cryptography.fernet import Fernet
from cryptography.hazmat.primitives import hashes
from cryptography.hazmat.primitives.kdf.pbkdf2 import PBKDF2HMAC


# Environment variable for encryption key (preferred method)
ENCRYPTION_KEY_ENV_VAR = "ARCVault_ENCRYPTION_KEY"

# Fallback: derive key from system properties (less secure but better than plain text)
_SYSTEM_SALT = b"ArcVault_Credential_Encryption_Salt_v1"


def _get_encryption_key() -> bytes:
    """
    Get encryption key from environment variable or derive from system.
    
    For production, set ARCVault_ENCRYPTION_KEY environment variable
    with a base64-encoded Fernet key (generate with Fernet.generate_key()).
    """
    # Try environment variable first (most secure)
    key_str = os.getenv(ENCRYPTION_KEY_ENV_VAR)
    if key_str:
        try:
            return base64.urlsafe_b64decode(key_str.encode())
        except Exception:
            # Invalid key format, fall back to derivation
            pass
    
    # Fallback: derive key from system properties
    # This is less secure but better than storing plain text
    # In production, always set ARCVault_ENCRYPTION_KEY
    import platform
    import getpass
    
    system_id = f"{platform.node()}{getpass.getuser()}".encode()
    kdf = PBKDF2HMAC(
        algorithm=hashes.SHA256(),
        length=32,
        salt=_SYSTEM_SALT,
        iterations=100000,
    )
    key = base64.urlsafe_b64encode(kdf.derive(system_id))
    return key


def encrypt_password(password: str) -> str:
    """Encrypt a password and return base64-encoded encrypted string."""
    if not password:
        return ""
    key = _get_encryption_key()
    fernet = Fernet(key)
    encrypted = fernet.encrypt(password.encode())
    return base64.urlsafe_b64encode(encrypted).decode()


def decrypt_password(encrypted_password: str) -> str:
    """Decrypt a base64-encoded encrypted password."""
    if not encrypted_password:
        return ""
    try:
        key = _get_encryption_key()
        fernet = Fernet(key)
        encrypted_bytes = base64.urlsafe_b64decode(encrypted_password.encode())
        decrypted = fernet.decrypt(encrypted_bytes)
        return decrypted.decode()
    except Exception as e:
        # If decryption fails, assume it's plain text (backward compatibility)
        # Log warning in production
        return encrypted_password


def generate_encryption_key() -> str:
    """
    Generate a new encryption key for use in ARCVault_ENCRYPTION_KEY.
    
    Returns base64-encoded key string.
    """
    key = Fernet.generate_key()
    return base64.urlsafe_b64encode(key).decode()

