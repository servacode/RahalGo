#!/usr/bin/env bash
# يبني ساعي رحّال غو للويندوز في dist/ — البرنامج الخفيّ ونسخةُ الأوامر، ومعهما ملفّاتُ التنصيب.
set -euo pipefail
cd "$(dirname "$0")"
rm -rf dist && mkdir -p dist
GOOS=windows GOARCH=amd64 go build -trimpath -ldflags "-s -w -H=windowsgui" -o dist/RahalGoSync.exe .
GOOS=windows GOARCH=amd64 go build -trimpath -ldflags "-s -w" -o dist/RahalGoSync-console.exe .
cp package/* dist/
echo "dist/ جاهز"
