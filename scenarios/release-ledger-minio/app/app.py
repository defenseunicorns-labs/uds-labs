# Copyright 2026 Defense Unicorns
# SPDX-License-Identifier: AGPL-3.0-or-later
from __future__ import annotations

import hashlib
import os
import re
import time
import uuid
from datetime import datetime, timezone
from pathlib import Path
from typing import Any

from flask import Flask, Response, jsonify, render_template_string, request


MAX_UPLOAD_BYTES = 1024 * 1024
SCHEMA = """
CREATE TABLE IF NOT EXISTS releases (
    id TEXT PRIMARY KEY,
    application_name TEXT NOT NULL,
    version TEXT NOT NULL,
    environment TEXT NOT NULL,
    created_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS evidence (
    id TEXT PRIMARY KEY,
    release_id TEXT NOT NULL REFERENCES releases(id) ON DELETE CASCADE,
    filename TEXT NOT NULL,
    object_key TEXT NOT NULL UNIQUE,
    size INTEGER NOT NULL,
    sha256 TEXT NOT NULL,
    created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS evidence_release_id_idx ON evidence(release_id);
"""


class MemoryMetadataStore:
    def __init__(self) -> None:
        self.releases: dict[str, dict[str, Any]] = {}
        self.evidence: dict[str, dict[str, Any]] = {}

    def create_release(self, values: dict[str, str]) -> dict[str, Any]:
        release = {
            "id": str(uuid.uuid4()),
            **values,
            "created_at": now(),
        }
        self.releases[release["id"]] = release
        return release

    def list_releases(self) -> list[dict[str, Any]]:
        return list(self.releases.values())

    def get_release(self, release_id: str) -> dict[str, Any] | None:
        return self.releases.get(release_id)

    def create_evidence(self, values: dict[str, Any]) -> dict[str, Any]:
        evidence = {"id": str(uuid.uuid4()), **values, "created_at": now()}
        self.evidence[evidence["id"]] = evidence
        return evidence

    def list_evidence(self, release_id: str) -> list[dict[str, Any]]:
        return [item for item in self.evidence.values() if item["release_id"] == release_id]

    def get_evidence(self, evidence_id: str) -> dict[str, Any] | None:
        return self.evidence.get(evidence_id)

    def delete_evidence(self, evidence_id: str) -> None:
        self.evidence.pop(evidence_id, None)


class PostgresMetadataStore:
    def __init__(self, settings: dict[str, str]) -> None:
        try:
            import psycopg
        except ImportError as error:  # pragma: no cover - image dependency error
            raise RuntimeError("psycopg is required for STORAGE_MODE=postgres") from error

        self.psycopg = psycopg
        self.settings = settings
        self.connection_string = {
            "host": settings["DATABASE_HOST"],
            "port": settings.get("DATABASE_PORT", "5432"),
            "dbname": settings["DATABASE_NAME"],
            "user": settings["DATABASE_USER"],
            "password": settings["DATABASE_PASSWORD"],
        }
        self.initialize()

    def connect(self):
        return self.psycopg.connect(**self.connection_string)

    def initialize(self) -> None:
        deadline = time.monotonic() + float(self.settings.get("DATABASE_STARTUP_TIMEOUT", "60"))
        while True:
            try:
                with self.connect() as connection:
                    with connection.cursor() as cursor:
                        for statement in SCHEMA.split(";"):
                            if statement.strip():
                                cursor.execute(statement)
                return
            except self.psycopg.OperationalError:
                if time.monotonic() >= deadline:
                    raise
                time.sleep(1)

    def create_release(self, values: dict[str, str]) -> dict[str, Any]:
        release = {"id": str(uuid.uuid4()), **values, "created_at": now()}
        with self.connect() as connection:
            with connection.cursor() as cursor:
                cursor.execute(
                    "INSERT INTO releases (id, application_name, version, environment, created_at) "
                    "VALUES (%(id)s, %(application_name)s, %(version)s, %(environment)s, %(created_at)s)",
                    release,
                )
        return release

    def list_releases(self) -> list[dict[str, Any]]:
        with self.connect() as connection:
            with connection.cursor() as cursor:
                cursor.execute(
                    "SELECT id, application_name, version, environment, created_at "
                    "FROM releases ORDER BY created_at DESC"
                )
                return [dict(zip(("id", "application_name", "version", "environment", "created_at"), row)) for row in cursor.fetchall()]

    def get_release(self, release_id: str) -> dict[str, Any] | None:
        with self.connect() as connection:
            with connection.cursor() as cursor:
                cursor.execute(
                    "SELECT id, application_name, version, environment, created_at "
                    "FROM releases WHERE id = %s",
                    (release_id,),
                )
                row = cursor.fetchone()
                return dict(zip(("id", "application_name", "version", "environment", "created_at"), row)) if row else None

    def create_evidence(self, values: dict[str, Any]) -> dict[str, Any]:
        evidence = {"id": str(uuid.uuid4()), **values, "created_at": now()}
        with self.connect() as connection:
            with connection.cursor() as cursor:
                cursor.execute(
                    "INSERT INTO evidence (id, release_id, filename, object_key, size, sha256, created_at) "
                    "VALUES (%(id)s, %(release_id)s, %(filename)s, %(object_key)s, %(size)s, %(sha256)s, %(created_at)s)",
                    evidence,
                )
        return evidence

    def list_evidence(self, release_id: str) -> list[dict[str, Any]]:
        with self.connect() as connection:
            with connection.cursor() as cursor:
                cursor.execute(
                    "SELECT id, release_id, filename, object_key, size, sha256, created_at "
                    "FROM evidence WHERE release_id = %s ORDER BY created_at DESC",
                    (release_id,),
                )
                keys = ("id", "release_id", "filename", "object_key", "size", "sha256", "created_at")
                return [dict(zip(keys, row)) for row in cursor.fetchall()]

    def get_evidence(self, evidence_id: str) -> dict[str, Any] | None:
        with self.connect() as connection:
            with connection.cursor() as cursor:
                cursor.execute(
                    "SELECT id, release_id, filename, object_key, size, sha256, created_at "
                    "FROM evidence WHERE id = %s",
                    (evidence_id,),
                )
                row = cursor.fetchone()
                keys = ("id", "release_id", "filename", "object_key", "size", "sha256", "created_at")
                return dict(zip(keys, row)) if row else None

    def delete_evidence(self, evidence_id: str) -> None:
        with self.connect() as connection:
            with connection.cursor() as cursor:
                cursor.execute("DELETE FROM evidence WHERE id = %s", (evidence_id,))


class FileContentStore:
    def __init__(self, directory: str) -> None:
        self.directory = Path(directory)
        self.directory.mkdir(parents=True, exist_ok=True)

    def put(self, object_key: str, content: bytes) -> None:
        destination = self.directory / object_key
        destination.parent.mkdir(parents=True, exist_ok=True)
        destination.write_bytes(content)

    def get(self, object_key: str) -> bytes | None:
        destination = self.directory / object_key
        return destination.read_bytes() if destination.is_file() else None

    def delete(self, object_key: str) -> None:
        destination = self.directory / object_key
        if destination.is_file():
            destination.unlink()


class MinioContentStore:
    def __init__(self, settings: dict[str, str]) -> None:
        try:
            from minio import Minio
            from minio.error import S3Error
        except ImportError as error:  # pragma: no cover - image dependency error
            raise RuntimeError("minio is required for EVIDENCE_STORAGE=minio") from error

        self.S3Error = S3Error
        endpoint = settings["S3_ENDPOINT"].removeprefix("http://").removeprefix("https://")
        self.client = Minio(
            endpoint,
            access_key=settings["S3_ACCESS_KEY"],
            secret_key=settings["S3_SECRET_KEY"],
            secure=settings.get("S3_USE_SSL", "false").lower() == "true",
        )
        self.bucket = settings["S3_BUCKET"]
        self.ensure_bucket()

    def ensure_bucket(self) -> None:
        if not self.client.bucket_exists(self.bucket):
            self.client.make_bucket(self.bucket)

    def put(self, object_key: str, content: bytes) -> None:
        from io import BytesIO

        self.client.put_object(self.bucket, object_key, BytesIO(content), len(content))

    def get(self, object_key: str) -> bytes | None:
        try:
            response = self.client.get_object(self.bucket, object_key)
            try:
                return response.read()
            finally:
                response.close()
                response.release_conn()
        except self.S3Error as error:
            if error.code == "NoSuchKey":
                return None
            raise

    def delete(self, object_key: str) -> None:
        self.client.remove_object(self.bucket, object_key)


HTML = """<!doctype html>
<html><head><title>Release Ledger</title></head>
<body><h1>Release Ledger</h1>
<p>Record releases and the evidence associated with each deployment.</p>
<p>Use the JSON API at <code>/api/releases</code> to create a release.</p>
</body></html>"""


def now() -> str:
    return datetime.now(timezone.utc).isoformat()


def create_metadata_store(settings: dict[str, str]):
    if settings.get("STORAGE_MODE", "memory") == "postgres":
        return PostgresMetadataStore(settings)
    return MemoryMetadataStore()


def create_content_store(settings: dict[str, str]):
    if settings.get("EVIDENCE_STORAGE", "filesystem") == "minio":
        return MinioContentStore(settings)
    # The stateless scenario runs as a non-root container. Keep the default
    # filesystem backend writable; its contents are intentionally ephemeral.
    return FileContentStore(settings.get("EVIDENCE_DIR", "/tmp/release-ledger/evidence"))


def create_app(test_config: dict[str, Any] | None = None) -> Flask:
    settings = dict(os.environ)
    if test_config:
        settings.update({key: str(value) for key, value in test_config.items()})

    app = Flask(__name__)
    metadata = create_metadata_store(settings)
    content = create_content_store(settings)
    app.config["MAX_CONTENT_LENGTH"] = MAX_UPLOAD_BYTES

    @app.get("/")
    def index():
        return render_template_string(HTML)

    @app.get("/health")
    def health():
        return jsonify(status="ok")

    @app.post("/api/releases")
    def create_release():
        payload = request.get_json(silent=True) or {}
        required = ("application_name", "version", "environment")
        if any(not isinstance(payload.get(key), str) or not payload[key].strip() for key in required):
            return jsonify(error="application_name, version, and environment are required"), 400
        return jsonify(metadata.create_release({key: payload[key].strip() for key in required})), 201

    @app.get("/api/releases")
    def list_releases():
        return jsonify(releases=metadata.list_releases())

    @app.get("/api/releases/<release_id>")
    def get_release(release_id: str):
        release = metadata.get_release(release_id)
        if release is None:
            return jsonify(error="release not found"), 404
        return jsonify(release)

    @app.post("/api/releases/<release_id>/evidence")
    def upload_evidence(release_id: str):
        if metadata.get_release(release_id) is None:
            return jsonify(error="release not found"), 404
        uploaded = request.files.get("file")
        if uploaded is None or not uploaded.filename:
            return jsonify(error="a file field is required"), 400
        filename = Path(uploaded.filename).name
        data = uploaded.read(MAX_UPLOAD_BYTES + 1)
        if len(data) > MAX_UPLOAD_BYTES:
            return jsonify(error="file exceeds 1 MiB limit"), 413
        evidence_id = str(uuid.uuid4())
        object_key = f"{release_id}/{evidence_id}-{safe_filename(filename)}"
        record = {
            "release_id": release_id,
            "filename": filename,
            "object_key": object_key,
            "size": len(data),
            "sha256": hashlib.sha256(data).hexdigest(),
        }
        content.put(object_key, data)
        try:
            result = metadata.create_evidence({"id": evidence_id, **record})
        except Exception:
            content.delete(object_key)
            raise
        return jsonify(result), 201

    @app.get("/api/releases/<release_id>/evidence")
    def list_evidence(release_id: str):
        if metadata.get_release(release_id) is None:
            return jsonify(error="release not found"), 404
        return jsonify(evidence=metadata.list_evidence(release_id))

    @app.get("/api/evidence/<evidence_id>/content")
    def download_evidence(evidence_id: str):
        evidence = metadata.get_evidence(evidence_id)
        if evidence is None:
            return jsonify(error="evidence not found"), 404
        data = content.get(evidence["object_key"])
        if data is None:
            return jsonify(error="evidence content not found"), 404
        return Response(data, mimetype="application/octet-stream")

    @app.delete("/api/evidence/<evidence_id>")
    def delete_evidence(evidence_id: str):
        evidence = metadata.get_evidence(evidence_id)
        if evidence is None:
            return jsonify(error="evidence not found"), 404
        content.delete(evidence["object_key"])
        metadata.delete_evidence(evidence_id)
        return "", 204

    return app


def safe_filename(filename: str) -> str:
    cleaned = re.sub(r"[^A-Za-z0-9._-]", "-", filename).strip(".")
    return cleaned or "evidence"


app = create_app()

if __name__ == "__main__":
    app.run(host="0.0.0.0", port=int(os.environ.get("PORT", "8080")))
