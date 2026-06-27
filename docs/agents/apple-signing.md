# Agent: `apple-signing`

**Phase A.** Owns the Apple Developer portal plumbing: App ID, bundle identifier,
capabilities, provisioning profiles, and TestFlight setup. This is what lets the `ui`
agent's app build to a real Vision Pro and ship to testers.

## Read first
- [`../../ORCHESTRATION.md`](../../ORCHESTRATION.md)
- [`../contracts/shared-constants.md`](../contracts/shared-constants.md) ← bundle id, platform, min OS

## Dependencies
- **Blocks:** `ui` *only* for building to real hardware / TestFlight. `ui` can scaffold and
  run in the simulator without you, so don't block them — get the App ID registered early.
- **Blocked by:** nothing, but **much of this needs the human owner's interactive login /
  2FA**. Your job is to prepare everything and hand the owner an exact click-by-click list.

## Important: what an agent can vs cannot do here
- ✅ Can: write `fastlane/` config, an `ExportOptions.plist`, document every step, prepare
  a `.xcconfig` with the team id / bundle id, write scripts that call `xcrun`/`altool`.
- ❌ Cannot: log into developer.apple.com with 2FA, accept program agreements, or click
  through the portal. **Flag these clearly for the owner.**

## Task checklist
1. **Confirm account details with the owner** — Apple Team ID, the Apple ID email, and
   whether an App Store Connect API key exists (preferred for automation; the owner can
   create one at App Store Connect → Users and Access → Integrations → keys).
2. **Register the App ID** for `org.richardmch.visionspots` (from `shared-constants.md`),
   platform **visionOS**. Document exact portal clicks for the owner if no API key.
3. **Capabilities** — enable what v1 needs. Likely just *Associated Domains* later (for the
   `vision-spots.richardmch.org` callback) and nothing exotic. Keep it minimal; expand on
   request from `ui`/`spotify-connection`.
4. **Signing config** — add a committed `app/Signing.xcconfig` (team id, bundle id, code
   sign style = Automatic to start) so the Xcode project the `ui` agent creates can
   reference it. No private keys or profiles committed.
5. **Provisioning** — prefer Xcode-managed automatic signing for development. For
   TestFlight, prepare a `fastlane/` setup (`Appfile`, `Fastfile` with a `beta` lane using
   `pilot`/`deliver`) and `app/ExportOptions.plist`. Document the API-key env vars.
6. **TestFlight** — create the App Store Connect app record (or document the steps), set up
   internal testing, and write `docs/release-checklist.md` for cutting a TestFlight build.
7. **Hand-off doc** — `docs/apple-setup.md`: exactly what the owner must click/run, in
   order, with screenshots-by-description, plus how `ui` consumes `Signing.xcconfig`.

## Definition of done
- App ID `org.richardmch.visionspots` registered for visionOS (or a precise checklist the
  owner can complete in <10 min).
- `app/Signing.xcconfig`, `app/ExportOptions.plist`, and `fastlane/` committed and
  documented — `ui` can set its target's config file and build/sign without guessing.
- `docs/apple-setup.md` + `docs/release-checklist.md` written.
- PR opened from `agent/apple-signing`; tell `ui` the bundle id + signing config are ready.
