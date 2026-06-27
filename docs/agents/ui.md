# Agent: `ui`

**Phase B.** Owns `app/` — the native visionOS SwiftUI client. **The human owner is the
most hands-on with this agent**, so optimize for readable, idiomatic SwiftUI and small
reviewable PRs. Expect the owner to iterate on design with you directly.

## Read first
- [`../../ORCHESTRATION.md`](../../ORCHESTRATION.md)
- [`../contracts/shared-constants.md`](../contracts/shared-constants.md) ← bundle id, scheme, backend URL
- [`../contracts/api-contract.md`](../contracts/api-contract.md) ← what the backend gives you

## Dependencies
- **Blocked by (hardware only):** `apple-signing` for building to a real Vision Pro /
  TestFlight. You do **not** need to wait for it — start now in the **visionOS simulator**
  with mock data.
- **Integrates with:** `spotify-connection` (Phase C) wires your views to real data. Until
  then, build against a mock data layer behind a protocol so the swap is trivial.

## Setup
```bash
git worktree add ../../.claude/worktrees/ui -b agent/ui
cd ../../.claude/worktrees/ui/app
```
Create the visionOS app (Xcode → visionOS → App, or via `xcodegen`/`tuist` if you prefer a
spec-driven project the owner can diff). Set the bundle id and reference
`Signing.xcconfig` from `apple-signing` once available.

## Architecture guidance
- **SwiftUI + visionOS idioms**: windows, volumes, and ornaments where they fit. Use a
  `NavigationSplitView`-style layout for library/playlists, glassy materials, and depth
  appropriate to Vision Pro. Don't over-engineer 3D for v1 — a great flat-ish spatial UI first.
- **State**: `@Observable` models, async/await networking, no third-party arch frameworks.
- **Data layer behind a protocol** so mock ⇄ real backend is one line:
  ```swift
  protocol SpotifyService { func me() async throws -> UserProfile; func playlists() async throws -> [Playlist]; ... }
  struct MockSpotifyService: SpotifyService { /* canned data for design iteration */ }
  struct LiveSpotifyService: SpotifyService { /* calls https://vision-spots.richardmch.org per api-contract */ }
  ```
- **Auth UX**: a "Connect Spotify" screen that launches `ASWebAuthenticationSession` to
  `GET /auth/start`'s `authorize_url`, handles the `visionspots://callback?session=...`
  redirect, and stores the session token in **Keychain**. (The `spotify-connection` agent
  owns the wiring details; you own the screens and the Keychain helper.)

## Task checklist
1. Create the visionOS Xcode project in `app/` with bundle id `org.richardmch.visionspots`.
2. Define the `SpotifyService` protocol + `MockSpotifyService` with rich canned data.
3. Build the core screens against mocks:
   - **Connect / sign-in** screen (the real OAuth launch comes from `spotify-connection`).
   - **Home / Library** (playlists, saved albums, recently played).
   - **Playlist / Album detail** with track lists.
   - **Search**.
   - **Now Playing** with Spotify Connect device picker + transport controls (play/pause/
     next/prev) — these map to `/api/player/*` in the contract.
4. Register the `visionspots://` URL scheme + Keychain session storage helper.
5. Polish: loading/empty/error states, async image loading, app icon placeholder.
6. `app/README.md`: how to open the project, switch between Mock and Live services, and run.

## Definition of done
- App builds and runs in the visionOS simulator against `MockSpotifyService`, with all core
  screens navigable.
- A single, documented switch flips Mock → Live (ready for `spotify-connection`).
- PR opened from `agent/ui`. Keep PRs small — the owner will review UI closely.
