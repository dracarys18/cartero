#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")"

name=$(sed -n 's/^versionName=//p' ../gradle.properties)
code=$(sed -n 's/^versionCode=//p' ../gradle.properties)
team=${APPLE_TEAM_ID:-$(sed -n 's/^TEAM_ID=//p' Configuration/Local.xcconfig)}
key_path=${ASC_KEY_PATH:-$(ls ~/.cartero/AuthKey_*.p8 | head -1)}
key_id=${ASC_KEY_ID:-$(basename "$key_path" .p8 | sed 's/^AuthKey_//')}
issuer=${ASC_ISSUER_ID:-$(cat ~/.cartero/asc-issuer-id)}
work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT

xcodebuild archive \
  -project iosApp.xcodeproj \
  -scheme iosApp \
  -configuration Release \
  -destination 'generic/platform=iOS' \
  -archivePath "$work/Cartero.xcarchive" \
  TEAM_ID="$team" \
  MARKETING_VERSION="$name" \
  CURRENT_PROJECT_VERSION="$code" \
  CODE_SIGNING_ALLOWED=NO

cat > "$work/ExportOptions.plist" <<PLIST
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>method</key>
  <string>app-store-connect</string>
  <key>destination</key>
  <string>upload</string>
  <key>teamID</key>
  <string>$team</string>
  <key>signingStyle</key>
  <string>automatic</string>
</dict>
</plist>
PLIST

PATH=/usr/bin:/bin:/usr/sbin:/sbin xcodebuild -exportArchive \
  -archivePath "$work/Cartero.xcarchive" \
  -exportOptionsPlist "$work/ExportOptions.plist" \
  -exportPath "$work/export" \
  -allowProvisioningUpdates \
  -authenticationKeyPath "$key_path" \
  -authenticationKeyID "$key_id" \
  -authenticationKeyIssuerID "$issuer"

echo "Uploaded Cartero $name ($code) to TestFlight"
