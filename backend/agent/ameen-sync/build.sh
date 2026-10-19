#!/usr/bin/env bash
# يبني برنامج مزامنة رحّال غو للويندوز — ملفٌّ واحدٌ بلا نافذة (dist/RahalGoSync.exe).
# والخادمُ يبنيه كذلك في صورته (Dockerfile) ويضيف رمزَ المتجر عند التحميل.
set -euo pipefail
cd "$(dirname "$0")"
rm -rf dist && mkdir -p dist
GOOS=windows GOARCH=amd64 go build -trimpath -ldflags "-s -w -H=windowsgui ${VERSION:+-X main.Version=$VERSION}" -o dist/RahalGoSync.exe .
echo "dist/RahalGoSync.exe ready"
