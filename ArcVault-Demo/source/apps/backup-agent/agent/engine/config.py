from typing import List, Optional, Any

from pydantic import BaseModel, Field, field_validator, model_validator


class BackupOptions(BaseModel):
    include_empty_dirs: bool = Field(default=True, description="/E")
    verbose: bool = Field(default=True, description="/V")
    report_extras: bool = Field(default=True, description="/X")
    retries: int = Field(default=1, ge=0, le=1000, description="/R:n")
    wait_seconds: int = Field(default=5, ge=0, le=60, description="/W:n")
    threads: int = Field(default=12, ge=1, le=128, description="/MT:n")
    copy_symlinks: bool = Field(default=False, description="/SL")
    remove_system_hidden: bool = Field(default=True, description="/A-:SH")


class BackupConfig(BaseModel):
    sources: List[str] = Field(min_length=1)
    destination: str
    options: BackupOptions = Field(default_factory=BackupOptions)
    source_credentials: Optional["Credentials"] = None
    destination_credentials: Optional["Credentials"] = None

    @field_validator("sources")
    @classmethod
    def _strip_sources(cls, v: List[str]) -> List[str]:
        cleaned = [s.strip() for s in v if s and s.strip()]
        if not cleaned:
            raise ValueError("At least one non-empty source is required")
        return cleaned

    @field_validator("destination")
    @classmethod
    def _strip_dest(cls, v: str) -> str:
        v = (v or "").strip()
        if not v:
            raise ValueError("Destination is required")
        return v


class Credentials(BaseModel):
    """
    Secure credentials model with encrypted password storage.
    
    Passwords are automatically encrypted when stored and decrypted when accessed.
    The password is stored encrypted internally but accessed transparently.
    """
    username: str
    password: str  # This will store encrypted password internally
    domain: Optional[str] = None
    
    @model_validator(mode="before")
    @classmethod
    def _encrypt_password_on_input(cls, data: Any) -> Any:
        """Encrypt password when credentials are created from plain text."""
        if isinstance(data, dict) and "password" in data:
            password_value = data["password"]
            if password_value:
                # Check if already encrypted (heuristic: long base64-like string)
                is_encrypted = (
                    len(password_value) > 50 and 
                    all(c in "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_=" for c in password_value)
                )
                if not is_encrypted:
                    # Import here to avoid circular imports
                    from .credential_encryption import encrypt_password
                    data["password"] = encrypt_password(password_value)
        return data
    
    def __getattribute__(self, name: str) -> Any:
        """Intercept password access to decrypt transparently."""
        if name == "password":
            # Access encrypted value directly from instance dict to avoid recursion
            instance_dict = object.__getattribute__(self, "__dict__")
            encrypted = instance_dict.get("password", "")
            if encrypted:
                from .credential_encryption import decrypt_password
                try:
                    return decrypt_password(encrypted)
                except Exception:
                    # If decryption fails, return as-is (backward compatibility)
                    return encrypted
            return encrypted
        return super().__getattribute__(name)



