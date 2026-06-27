#!/usr/bin/env bash
# Regenerate the Xcode project from project.yml.
#
# The .xcodeproj is generated (gitignored) — project.yml is the source of truth.
# Run this after cloning or after editing project.yml:  ./regen.sh
set -euo pipefail
cd "$(dirname "$0")"

if ! command -v xcodegen >/dev/null 2>&1; then
  echo "XcodeGen not found. Install with: brew install xcodegen" >&2
  exit 1
fi

xcodegen generate

# XcodeGen emits objectVersion 77 (Xcode 16.3+). Older Xcode (e.g. 16.0) can't open that
# format. Pin to 56, which every Xcode 15/16 reads. Harmless on newer Xcode.
sed -i '' 's/objectVersion = 77;/objectVersion = 56;/' VisionSpots.xcodeproj/project.pbxproj

echo "Generated VisionSpots.xcodeproj — open it with: xed ."
