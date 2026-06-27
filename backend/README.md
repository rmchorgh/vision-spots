# backend

Go OAuth/token service for vision-spots (PKCE + session JWT broker).

See `../docs/agents/backend.md` for full task spec and `../docs/contracts/api-contract.md` for the exact endpoints this service must implement.

## Quick start

```bash
cp .env.example .env
# edit .env with your Spotify credentials + SESSION_SIGNING_KEY
make run
```

Then open `https://vision-spots.richardmch.org/auth/start` (or use the app).

## OAuth flow (heavily commented in code)

1. `/auth/start` → generates PKCE verifier + S256 challenge + random `state`. Stores verifier by `state` (in-memory map + mutex for v1).
2. App opens Spotify authorize URL in `ASWebAuthenticationSession`.
3. Spotify redirects to backend `/callback?code=...&state=...`.
4. Backend exchanges code+verifier for tokens, creates signed session JWT, redirects to `visionspots://callback?session=<jwt>`.
5. App uses the session JWT in `Authorization: Bearer <jwt>` for all subsequent calls.
6. Backend proxies `/api/spotify/*` and player endpoints, auto-refreshing Spotify token on 401.

**Token storage note**: In-memory for single-user v1. See `internal/session/store.go` for where to plug in Redis/SQLite later.

**Security**: Client secret never leaves server. Session JWT is short-lived and signed with `SESSION_SIGNING_KEY`. Tokens are redacted in logs.

## Development

- `make run` — starts server on $PORT (default 5055)
- `make test` — runs all tests
- `make docker-build` — builds production image
- `make clean` — removes binaries

Full implementation lives in small internal packages:
- `internal/config`
- `internal/spotify`
- `internal/session`
- `internal/httpapi`

All exported functions are heavily commented so the owner can follow the OAuth/PKCE flow easily.
