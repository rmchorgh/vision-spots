# API contract — app ↔ backend

This is the interface between the visionOS app (`ui` + `spotify-connection`) and the Go
service (`backend`). Both sides build against this document so they can be developed in
parallel. Change it via PR and notify the other side.

## Why a backend at all?

PKCE alone can authenticate a public client, but we route through the backend so that:
- the **client secret never ships in the app**,
- token **refresh** happens server-side on a schedule,
- the app holds only a short-lived **vision-spots session token**, not raw Spotify tokens.

## The auth flow (Authorization Code + PKCE, brokered)

```
app                         backend                         spotify
 │  1. GET /auth/start ───────▶│                               │
 │  ◀── {authorize_url, state} │  (backend builds PKCE         │
 │                             │   challenge, stores verifier) │
 │  2. open authorize_url in ASWebAuthenticationSession ──────▶│
 │                             │      user approves            │
 │                             │◀── 302 to /callback?code&state│
 │  3. spotify ─▶ backend /callback (code,state)              │
 │                             │── exchange code+verifier ────▶│
 │                             │◀──── access+refresh tokens ───│
 │  ◀── 302 visionspots://callback?session=<jwt> ─────────────│
 │  4. app stores session jwt in Keychain                     │
 │  5. app calls backend API with Authorization: Bearer <jwt> │
```

## Endpoints

All JSON. App authenticates to the backend with `Authorization: Bearer <session-jwt>`
except where noted.

### `GET /auth/start`  *(no auth)*
Begins login. Backend creates a PKCE verifier/challenge + `state`, stashes them.
```json
200 → { "authorize_url": "https://accounts.spotify.com/authorize?...", "state": "..." }
```

### `GET /callback?code&state`  *(no auth; Spotify redirects here)*
Backend validates `state`, exchanges `code` + verifier for Spotify tokens, stores them
keyed by a new session id, then redirects the user agent back to the app:
```
302 → visionspots://callback?session=<jwt>
```

### `POST /auth/refresh`  *(Bearer session)*
Returns a fresh session jwt if the underlying Spotify token was refreshed. App calls on
401 or on launch.
```json
200 → { "session": "<jwt>", "expires_in": 3600 }
```

### `GET /me`  *(Bearer session)*
Proxy of Spotify `GET /v1/me` (so the app never sees the raw Spotify token).
```json
200 → { "id": "...", "display_name": "...", "product": "premium|free", "image": "..." }
```

### `GET /api/spotify/*`  *(Bearer session)*  — generic proxy
Backend forwards to `https://api.spotify.com/v1/*`, injecting the stored Spotify access
token. Example: `GET /api/spotify/me/playlists?limit=50`. Keeps the access token off the
device and centralizes refresh-on-401. The `spotify-connection` agent decides whether to
use this generic proxy or add typed endpoints (`/api/playlists`, `/api/search`, …).

### Playback control (Spotify Connect) *(Bearer session)*
Thin proxies over Spotify's player API:
- `GET  /api/player/devices`          → list Connect devices
- `PUT  /api/player/play`             → body `{ device_id, context_uri|uris, position_ms }`
- `PUT  /api/player/pause`
- `POST /api/player/next` / `previous`
- `GET  /api/player/state`            → currently playing + device

## Errors

```json
{ "error": "spotify_unauthorized" | "session_expired" | "premium_required" | "bad_request", "message": "human readable" }
```
- `session_expired` → app calls `/auth/refresh`, retries once, else re-runs `/auth/start`.
- `premium_required` → Connect playback control needs Spotify Premium; surface in UI.

## Open questions for the implementing agents
- Token storage on the backend: in-memory map is fine for a single-user v1; note in the
  backend task file whether to add SQLite/Redis for multi-user.
- Session jwt lifetime vs Spotify token lifetime — keep app session short, refresh silently.
