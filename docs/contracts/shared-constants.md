# Shared constants — the single source of truth

Every agent must use these exact values. If you think one needs to change, change it
**here first** and call it out in your PR so others re-sync. Do not hardcode divergent
values anywhere else.

## Identity

| Name              | Value                                  | Used by                          |
| ----------------- | -------------------------------------- | -------------------------------- |
| Project name      | `vision-spots`                         | everyone                         |
| Bundle identifier | `org.richardmch.visionspots`           | `apple-signing`, `ui`            |
| App display name  | `vision-spots`                         | `ui`                             |
| GitHub repo       | `rmchorgh/vision-spots` (public)       | everyone                         |

> Note: Apple bundle ids allow letters, numbers, hyphens and dots, but to avoid edge
> cases the id intentionally has **no hyphen** (`visionspots`, not `vision-spots`).

## URLs & networking

| Name                     | Value                                            | Used by                       |
| ------------------------ | ------------------------------------------------ | ----------------------------- |
| Backend public base URL  | `https://vision-spots.richardmch.org`            | `ui`, `spotify-connection`    |
| Subdomain (tunnel)       | `vision-spots`                                   | `selfhost-dns`                |
| Backend container port   | `5055`                                           | `backend`, `selfhost-dns`     |
| Spotify OAuth redirect   | `https://vision-spots.richardmch.org/callback`   | `backend`, `spotify-connection`, **set in Spotify dashboard** |
| App deep-link scheme     | `visionspots://callback`                         | `ui`, `spotify-connection`    |

> Port 5055 was chosen to avoid the self-host repo's existing ports (5000 api, 3000
> portfolio, 9000 storage, 3301 telemetry, 4000 teslamate).

## Spotify

| Name                  | Value / where it lives                                              |
| --------------------- | ------------------------------------------------------------------ |
| `SPOTIFY_CLIENT_ID`   | from developer.spotify.com dashboard → `backend/.env` (gitignored) |
| `SPOTIFY_CLIENT_SECRET` | from dashboard → `backend/.env` (gitignored), **backend only**    |
| OAuth scopes (v1)     | `user-read-private user-read-email user-library-read playlist-read-private user-read-playback-state user-modify-playback-state user-read-currently-playing streaming` |
| Auth flow             | Authorization Code **with PKCE**; client secret stays server-side  |

## Apple

| Name              | Value                                       |
| ----------------- | ------------------------------------------- |
| Team / account    | the owner's existing Apple Developer account |
| Platform target   | visionOS (Apple Vision Pro)                  |
| Min visionOS       | `2.0` (revisit if a needed API requires newer) |
| Distribution       | TestFlight first, App Store later            |

## Environment file template

`backend/.env.example` (committed) documents the shape; `backend/.env` (gitignored)
holds real values:

```
SPOTIFY_CLIENT_ID=
SPOTIFY_CLIENT_SECRET=
SPOTIFY_REDIRECT_URI=https://vision-spots.richardmch.org/callback
PORT=5055
SESSION_SIGNING_KEY=        # random 32+ bytes, base64; backend mints app session tokens
ALLOWED_ORIGIN=visionspots://callback
```
