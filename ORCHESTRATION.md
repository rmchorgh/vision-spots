# Orchestration — how the agent swarm builds vision-spots

This is the entry point for every agent. Read it fully before touching code.

## TL;DR for an agent picking up work

1. Find your task file in [`docs/agents/`](docs/agents/).
2. Read [`docs/contracts/shared-constants.md`](docs/contracts/shared-constants.md) and
   [`docs/contracts/api-contract.md`](docs/contracts/api-contract.md). **Never invent
   values that those files define** (bundle id, redirect URI, ports, endpoint shapes).
3. Create your worktree (see [Working in a worktree](#working-in-a-worktree)).
4. Do the ordered checklist in your task file. Open a PR when your "Definition of done" is met.

## Roles

| Agent                | Owns            | One-line job                                                        |
| -------------------- | --------------- | ------------------------------------------------------------------- |
| `ui`                 | `app/`          | Native visionOS SwiftUI client. **The human owner is hands-on here.** |
| `backend`            | `backend/`      | Go OAuth/token service: PKCE exchange, refresh, session tokens.     |
| `spotify-connection` | app + backend glue | Spotify OAuth from the app, Web API browse/search, Connect playback control. |
| `apple-signing`      | Apple Dev portal | App ID, bundle id, provisioning profiles, capabilities, TestFlight. |
| `selfhost-dns`       | the **self-host** repo | Add `vision-spots.richardmch.org` tunnel route + Docker service. |
| `research-playback`  | `docs/`         | Investigate replicating iPad-style **in-app** Spotify playback on Vision Pro. |

## Build order (dependency graph)

```
Phase A  (start in parallel, no inter-dependencies)
  ├─ backend ............... scaffolds Go service + OAuth endpoints + Dockerfile
  ├─ apple-signing ......... App ID + provisioning so the app can build to device/TestFlight
  └─ research-playback ..... pure research, writes findings to docs/, blocks nobody

Phase B  (start once its dependency lands)
  ├─ selfhost-dns .......... needs backend's Dockerfile + chosen port  (depends: backend)
  └─ ui .................... needs the bundle id                       (depends: apple-signing)
                            // ui can begin scaffolding with MOCK data the moment Phase A starts;
                            // it only truly needs apple-signing to build to real hardware.

Phase C  (integration)
  └─ spotify-connection .... needs backend deployed (via selfhost-dns) + app shell (from ui)
```

### Why this order

- **backend first** because both `selfhost-dns` (needs its container/port) and
  `spotify-connection` (needs its endpoints) depend on it.
- **apple-signing in parallel** because it's all Apple-portal work and only gates
  *running on real hardware*, not writing code.
- **ui can start immediately** against mock data, so the human owner isn't blocked.
- **spotify-connection last** because it's the wire-up that needs both ends to exist.

## Manual steps only the human owner can do

Agents must **stop and flag** these — they require interactive logins / dashboards:

1. **Register the Spotify app** at <https://developer.spotify.com/dashboard> →
   gives `SPOTIFY_CLIENT_ID` + `SPOTIFY_CLIENT_SECRET`. Set the redirect URI to the
   exact value in `shared-constants.md`. (Needed by `backend` + `spotify-connection`.)
2. **Apple Developer portal logins** — registering the App ID, devices, and creating
   provisioning profiles may need interactive auth / 2FA. (`apple-signing` will prepare
   everything and tell you exactly what to click.)
3. **DNS / Cloudflare** — `selfhost-dns` edits config in the self-host repo, but the
   human runs `bash cloudflare/install.sh` and restarts the tunnel on the Mac Mini.

Secrets go in `backend/.env` (gitignored) and the self-host repo — **never committed**.

## Working in a worktree

Each agent works in its own git worktree so the swarm never collides on `main`.

```bash
# from the repo root, pick a branch named after your agent
git worktree add .claude/worktrees/<agent> -b agent/<agent>
cd .claude/worktrees/<agent>
# ...do your work, commit on the agent/<agent> branch...
git push -u origin agent/<agent>
gh pr create --fill --base main
```

Rules:
- One agent = one worktree = one `agent/<name>` branch = one PR.
- Stay inside your "Owns" path. If you must touch a shared contract file, say so loudly
  in the PR description so other agents re-sync.
- `selfhost-dns` is the exception: its real work happens in the **separate self-host
  repo** at `../self-host`, not in a worktree of this repo. See its task file.

## Definition of done for the whole project (v1)

A Vision Pro user installs vision-spots, signs into Spotify, browses their library and
playlists, searches, and controls playback on a Spotify Connect device — all from a
native visionOS UI, with tokens brokered by the self-hosted Go service.
