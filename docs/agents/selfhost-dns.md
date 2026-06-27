# Agent: `selfhost-dns`

**Phase B.** Exposes the Go backend at `https://vision-spots.richardmch.org` by adding a
Cloudflare Tunnel route + Docker service **in the separate self-host repo**, following its
existing conventions.

## ⚠️ This work happens in a DIFFERENT repo
Your changes go in the self-host repo at **`../self-host`** (sibling of this repo:
`/Users/richard/Documents/projects/self-host`), *not* in a worktree of vision-spots.
That repo has its own git history, CLAUDE.md, and PR flow — read its `claude.md` first.

## Read first
- `../self-host/claude.md` (its architecture + "Workflow for adding services")
- `../self-host/cloudflare/config.yml`, `../self-host/cloudflare/install.sh`
- An existing service dir for the pattern, e.g. `../self-host/docker/api/` and its `tunnel.json`
- [`../contracts/shared-constants.md`](../contracts/shared-constants.md) ← subdomain + port 5055

## Dependencies
- **Blocked by:** `backend` — you need its `Dockerfile` and confirmed port (5055) so your
  compose service builds the right image. You can prep the tunnel route before that lands.
- **Blocks:** `spotify-connection` end-to-end testing (the app needs a live HTTPS backend).

## How the self-host repo works (summary)
- Each service is a dir `docker/<service>/` with a `compose.yml` and a `tunnel.json`
  (`{ subdomain, port }`, schema at `docker/tunnel.schema.json`).
- `cloudflare/install.sh` reads every `tunnel.json` and regenerates `cloudflare/config.yml`.
- The root `docker/compose.yml` aggregates services; systemd runs `docker compose up`.

## Task checklist (in the self-host repo)
1. `git -C ../self-host worktree add .claude/wt/vision-spots -b feat/vision-spots` (or just a
   branch — match that repo's convention; check its CLAUDE.md).
2. Create `docker/vision-spots/`:
   - `tunnel.json` → `{ "$schema": "../tunnel.schema.json", "subdomain": "vision-spots", "port": 5055 }`
   - `compose.yml` → a service that **builds from the GitHub repo**
     `github.com/rmchorgh/vision-spots` (path `backend/`), like the other services build
     from GitHub. Map container `5055` → host `5055`. Pass the env file / secrets the way
     this repo handles other services' secrets (check an existing service; **do not commit
     `SPOTIFY_CLIENT_SECRET`** — document where the owner puts it on the host).
3. Add the service to the root `docker/compose.yml` aggregator if that's how others are wired.
4. Run `bash cloudflare/install.sh` to regenerate `config.yml`; confirm a
   `hostname: vision-spots.richardmch.org → http://localhost:5055` ingress entry appears.
5. **Flag for the owner** (manual, on the Mac Mini): place the backend secrets, restart the
   tunnel + `docker compose up -d`, and confirm `https://vision-spots.richardmch.org/healthz`
   returns 200 from the public internet.
6. Document what you changed in the self-host repo's conventions (a short note in its PR).

## Definition of done
- `docker/vision-spots/{tunnel.json,compose.yml}` added; `cloudflare/config.yml` regenerated
  with the new ingress entry; root aggregator updated if applicable.
- No secrets committed; a clear owner runbook for bringing it up on the host.
- PR opened **in the self-host repo**; report back here that the subdomain is live (or the
  exact remaining manual step). Tell `spotify-connection` the backend URL is reachable.
