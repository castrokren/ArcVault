import os
from typing import List, Optional, Any, Tuple
from pathlib import Path
from uuid import uuid4

from pydantic import BaseModel, Field, field_validator, model_validator, computed_field, ConfigDict


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
    model_config = ConfigDict(validate_assignment=True)
    
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
    
    @field_validator('destination')
    @classmethod
    def validate_destination_writable(cls, v: str) -> str:
        """
        Validate that destination exists and is writable.
        
        Can be skipped by setting environment variable:
        BACKUP_SKIP_DESTINATION_CHECK=true
        
        This is useful for:
        - CI/CD pipelines where destination is mounted later
        - Config validation without actual filesystem access
        - Testing with mock destinations
        """
        # Allow skipping validation via environment variable
        if os.getenv("BACKUP_SKIP_DESTINATION_CHECK", "").lower() in ("true", "yes", "1"):
            return v
        
        path = Path(v)
        if not path.exists():
            raise ValueError(f"Destination does not exist: {v}")
        
        # Test write access
        test_file = path / f".write_test_{uuid4().hex}"
        try:
            test_file.touch()
            test_file.unlink()
        except Exception as e:
            raise ValueError(f"Destination not writable: {v} - {e}")
        return v
    
    def validate_destination_accessible(self, raise_on_error: bool = False) -> Tuple[bool, Optional[str]]:
        """
        Pre-flight check: Verify destination is currently accessible.
        
        Call this immediately before starting a backup to ensure the destination
        is available. This is separate from the initial validation because:
        - Network shares can be mounted/unmounted between config load and backup run
        - Destinations might be temporarily unavailable (maintenance, reboots)
        - Provides opportunity for retry logic or user notification
        
        Args:
            raise_on_error: If True, raises ValueError on failure instead of returning False
            
        Returns:
            Tuple of (is_accessible, error_message)
            - (True, None) if destination is accessible
            - (False, "error message") if destination is not accessible
            
        Raises:
            ValueError: If raise_on_error=True and destination is not accessible
            
        Example:
            >>> config = load_backup_config(...)
            >>> accessible, error = config.validate_destination_accessible()
            >>> if not accessible:
            >>>     logger.error(f"Destination not accessible: {error}")
            >>>     retry_later()
        """
        path = Path(self.destination)
        
        # Check if path exists
        if not path.exists():
            error_msg = f"Destination does not exist: {self.destination}"
            if raise_on_error:
                raise ValueError(error_msg)
            return False, error_msg
        
        # Test write access
        test_file = path / f".write_test_{uuid4().hex}"
        try:
            test_file.touch()
            test_file.unlink()
            return True, None
        except Exception as e:
            error_msg = f"Destination not writable: {self.destination} - {e}"
            if raise_on_error:
                raise ValueError(error_msg)
            return False, error_msg


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



