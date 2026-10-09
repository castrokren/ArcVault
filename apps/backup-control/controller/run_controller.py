import os
import sys
from pathlib import Path


def main() -> None:
    """Bootstrap to run the Streamlit controller app programmatically.

    This avoids relying on the `streamlit` CLI being on PATH and works well
    when bundled into a single-file binary (e.g., PyInstaller).
    """
    # Resolve the Streamlit app path next to this file
    app_path = Path(__file__).with_name("app.py")
    if not app_path.exists():
        sys.stderr.write(f"Controller app not found: {app_path}\n")
        sys.exit(1)

    # Runtime configuration via environment variables
    port_str = os.getenv("CONTROLLER_PORT", "8501")
    try:
        port = int(port_str)
    except ValueError:
        port = 8501
    address = os.getenv("CONTROLLER_ADDRESS", "0.0.0.0")

    # Prefer headless in service/daemon contexts
    headless = (os.getenv("STREAMLIT_HEADLESS", "1").strip().lower() in {"1", "true", "yes", "on"})

    # Suppress Streamlit usage prompts/telemetry in packaged mode
    os.environ.setdefault("STREAMLIT_BROWSER_GATHER_USAGE_STATS", "false")

    # Defer import to keep startup minimal and avoid side effects before checks
    from streamlit.web.bootstrap import run as st_run  # type: ignore

    flag_options = {
        "server.port": port,
        "server.address": address,
        "server.headless": headless,
    }

    # Launch Streamlit programmatically
    st_run(str(app_path), "", [], flag_options)


if __name__ == "__main__":
    main()


