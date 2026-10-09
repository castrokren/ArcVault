FROM python:3.11-slim

ENV PYTHONUNBUFFERED=1 \
    PIP_NO_CACHE_DIR=1 \
    STREAMLIT_BROWSER_GATHER_USAGE_STATS=false

WORKDIR /app

# Copy only the controller package to leverage Docker layer caching
COPY apps/backup-control /app/apps/backup-control

# Install controller dependencies
RUN pip install --upgrade pip \
    && pip install -e /app/apps/backup-control

# Default runtime configuration
ENV CONTROLLER_PORT=8501 \
    CONTROLLER_ADDRESS=0.0.0.0

# The controller reads these at runtime (mount /config with your keys)
ENV CTRL_JWT_PRIVATE_KEY_PATH=/config/controller_private.pem \
    AGENT_CA_BUNDLE=

EXPOSE 8501

CMD ["python", "/app/apps/backup-control/controller/run_controller.py"]


