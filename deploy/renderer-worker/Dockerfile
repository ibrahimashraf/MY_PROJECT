# Build stage
FROM python:3.12-slim-bookworm AS builder

ENV DEBIAN_FRONTEND=noninteractive
RUN apt-get update && apt-get install -y --no-install-recommends \
    build-essential \
    libpango-1.0-0 \
    libharfbuzz0b \
    libpangoft2-1.0-0 \
    libffi-dev \
    && rm -rf /var/lib/apt/lists/*

RUN pip install --no-cache-dir --prefix=/install \
    weasyprint==69.0 \
    segno>=1.6.1

# Production image
FROM python:3.12-slim-bookworm

ENV DEBIAN_FRONTEND=noninteractive
RUN apt-get update && apt-get install -y --no-install-recommends \
    libpango-1.0-0 \
    libharfbuzz0b \
    libpangoft2-1.0-0 \
    fonts-noto-core \
    fontconfig \
    && rm -rf /var/lib/apt/lists/*

COPY --from=builder /install /usr/local

# Create unprivileged service user and directories
RUN groupadd -g 10001 integin && \
    useradd -u 10001 -g integin -s /bin/false -d /nonexistent integin && \
    mkdir -p /app /tmp/renderer && \
    chown -R integin:integin /app /tmp/renderer

WORKDIR /app
COPY worker.py /app/worker.py

USER 10001:10001

# Enforce zero egress & read-only execution profile
ENTRYPOINT ["python3", "/app/worker.py"]
