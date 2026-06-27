# Agent: `spotify-connection`

**Phase C.** The integration agent. Wires the visionOS app's `LiveSpotifyService` to the
real backend, completes the OAuth handshake end-to-end, and implements browse/search/
Spotify-Connect playback against the live Spotify Web API.

## Read first
- [`../../ORCHESTRATION.md`](../../ORCHESTRATION.md)
- [`../contracts/shared-constants.md`](../contracts/shared-constants.md)
- [`../contracts/api-contract.md`](../contracts/api-contract.md) ← the interface you connect both ends of

## Dependencies
- **Blocked by:** `backend` (endpoints implemented) **and** `selfhost-dns` (backend live at
  `https://vision-spots.richardmch.org`) **and** `ui` (app shell + `SpotifyService`
  protocol + Connect screen). Also needs the human to have **registered the Spotify app**
  and set the redirect URI (see ORCHESTRATION → manual steps).
- **You may touch both `app/` and `backend/`** — you're the glue. Coordinate via PR notes;
  prefer additive changes and don't rewrite the other agents' structure.

## Setup
```bash
git worktree add ../../.claude/worktrees/spotify-connection -b agent/spotify-connection
cd ../../.claude/worktrees/spotify-connection
```

## Task checklist
1. **End-to-end auth.** Implement the app side of the flow in `api-contract.md`:
   - Launch `ASWebAuthenticationSession` with `authorize_url` from `GET /auth/start`.
   - Handle the `visionspots://callback?session=<jwt>` redirect; store jwt in Keychain.
   - Add a backend client that sends `Authorization: Bearer <jwt>` and transparently calls
     `POST /auth/refresh` on `session_expired`, retrying once.
   Verify the full dance against real Spotify with a real account.
2. **`LiveSpotifyService`.** Implement the protocol the `ui` agent defined, calling the
   backend proxy endpoints (`/me`, `/api/spotify/*` or typed endpoints) — map JSON to the
   app's models. Flip the app's default service from Mock → Live behind a build flag.
3. **Browse/search.** Library (playlists, saved albums, recently played), playlist/album
   detail, and search — paginated, with caching where sensible.
4. **Spotify Connect playback.** Device picker via `GET /api/player/devices`; transport via
   `/api/player/play|pause|next|previous`; live now-playing via `GET /api/player/state`
   (poll or refresh). Handle `premium_required` gracefully in the UI (Connect control needs
   Premium) — coordinate copy with the `ui` agent.
5. **Hardening.** Token-expiry edge cases, no-active-device state, rate limiting (429 +
   `Retry-After`), and offline/error states. Never log tokens.
6. **Backend typed endpoints (optional).** If the generic `/api/spotify/*` proxy is awkward,
   add typed endpoints in `backend/` (`/api/playlists`, `/api/search`, …) and update
   `api-contract.md` in the same PR.

## Definition of done
- A real Spotify account can sign in from the visionOS app and browse library + search with
  live data.
- Selecting a Spotify Connect device and pressing play/pause/next controls real playback.
- `premium_required` and `session_expired` paths are handled in-app.
- PR(s) opened from `agent/spotify-connection`; `api-contract.md` updated if the interface
  changed. This satisfies the project's v1 "Definition of done" in ORCHESTRATION.md.
