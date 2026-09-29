# Copyright 2026 Defense Unicorns
# SPDX-License-Identifier: AGPL-3.0-or-later
FROM python:3.11-slim
WORKDIR /app
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt
COPY app.py .
RUN useradd --uid 1000 --create-home appuser
ENV EVIDENCE_DIR=/var/lib/release-ledger/evidence
USER 1000
EXPOSE 8080
CMD ["python", "app.py"]
