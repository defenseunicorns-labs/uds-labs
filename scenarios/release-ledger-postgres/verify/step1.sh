#!/bin/bash
set -euo pipefail
cd /root/release-ledger-postgres
test -f app.py
grep -q 'DATABASE_USER' app.py
grep -q 'DATABASE_PASSWORD' app.py
grep -q 'STORAGE_MODE' app.py
echo 'Storage contract inspected'
