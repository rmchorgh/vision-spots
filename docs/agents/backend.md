# Agent: `backend`

**Phase A.** Owns `backend/`. Builds the Go auth/token service deployed at
`https://vision-spots.richardmch.org`. **Heavily comment the code** — the owner wants to
read it and understand how the OAuth flow works.

## Read first
- [`../../ORCHESTRATION.md`](../../ORCHESTRATION.md)
- [`../contracts/shared-constants.md`](../contracts/shared-constants.md)
- [`../contracts/api-contract.md`](../contracts/api-contract.md) ← you implement this exactly

## Dependencies
- **Blocks:** `selfhost-dns` (needs your Dockerfile + port 5055) and `spotify-connection`
  (needs your endpoints live).
- **Blocked by:** nothing. You can build and unit-test the whole flow with a fake Spotify
  before the real credentials exist. Flag the human to register the Spotify app (gives
  `SPOTIFY_CLIENT_ID`/`SECRET`) before end-to-end testing.

## Setup
```bash
git worktree add ../../.claude/worktrees/backend -b agent/backend
cd ../../.claude/worktrees/backend/backend
go mod init github.com/rmchorgh/vision-spots/backend
```

## Task checklist
1. **Project skeleton** in `backend/`:
   - `main.go`, `go.mod`, `internal/` packages, `Dockerfile`, `.env.example`,
     `README.md`, `Makefile` (`run`, `test`, `docker-build`).
   - Use the std lib `net/http` + a light router (chi is fine) — keep deps minimal.
2. **Config loading** — read `.env` / environment per `shared-constants.md`. Fail fast
   with a clear message if `SPOTIFY_CLIENT_ID`/`SECRET`/`SESSION_SIGNING_KEY` are missing.
3. **PKCE + state store** — generate `code_verifier`/`code_challenge` (S256) and a random
   `state`; store the verifier keyed by state with a short TTL. In-memory map with a mutex
   is fine for single-user v1 (note in README how to swap for Redis/SQLite later).
4. **Endpoints** — implement every endpoint in `api-contract.md`:
   `GET /auth/start`, `GET /callback`, `POST /auth/refresh`, `GET /me`,
   `GET /api/spotify/*` proxy, and the `/api/player/*` Connect controls.
5. **Spotify client** — `internal/spotify`: token exchange, refresh, and an authenticated
   `Do(req)` that injects the access token and **auto-refreshes once on 401**.
6. **Sessions** — mint a signed session token (JWT, HS256 with `SESSION_SIGNING_KEY`) that
   maps to a stored Spotify token set. Middleware validates `Authorization: Bearer`.
7. **Health + logging** — `GET /healthz` (used by Docker healthcheck), structured request
   logs, and a clear error JSON shape (`api-contract.md` → Errors).
8. **Dockerfile** — multi-stage, distroless/alpine final image, exposes `5055`, runs as
   non-root, `HEALTHCHECK` hitting `/healthz`. This is what `selfhost-dns` will reference.
9. **Tests** — table-driven tests for the PKCE helper, state TTL, and the refresh-on-401
   path using an `httptest` fake Spotify. `go test ./...` green.
10. **README** in `backend/` — how the flow works end to end, how to run locally
    (`make run` with a `.env`), and the curl commands to walk the auth dance.

## Coding standards (owner preference)
- **Comment generously.** Every exported function gets a doc comment; the OAuth/PKCE bits
  get inline comments explaining *why*, not just *what*. Assume the reader is learning OAuth.
- Small, named packages under `internal/` (`config`, `spotify`, `session`, `httpapi`).
- No secrets in code or logs. Redact tokens in any log line.

## Definition of done
- `go test ./...` passes; `docker build` produces a runnable image on port 5055.
- `make run` + a `.env` lets you complete the full login dance against real Spotify
  (after the human registers the app) and `GET /me` returns the profile.
- `backend/.env.example` and `backend/README.md` are committed; no real secret committed.
- PR opened from `agent/backend` to `main`; ping `selfhost-dns` that the Dockerfile is ready.
