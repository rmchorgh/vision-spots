# PR Guidelines

sEvery agent PR **must** follow this exactly.

## Structure

**Title:** `&lt;scope&gt;: &lt;one-line summary&gt;`
- scope: lowercase domain (e.g., `ui`, `backend`, `spotify-connection`)
- Capitalize major words, action verb start

**Body:**

1. **One-sentence overview** — what + why + context (agent/task reference).

2. **## What's here** (bulleted)
   - Key new pieces (structs/protocols/files/endpoints)
   - Bold critical seams (`**SpotifyService protocol**`)
   - Inline code for types/constants

3. **## The one switch** (if applicable)
   - Single flag/file for mock/live or feature toggle

4. **## Verification**
   ```
   # Exact shell commands to type-check/build/test
   npm run lint &amp;&amp; npm run typecheck
   swift build
   ```

5. **## Handoffs** (bulleted)
   - Next agent tasks/blockers
   - Manual human steps (secrets, portals)

## Rules
- Reference your `docs/agents/&lt;name&gt;.md` task file.
- **No secrets** — flag them for handoff.
- Link contracts touched: `[api-contract](docs/contracts/api-contract.md)`
- End with 🤖 **Generated with Opesncode/Claude Code and [MODEL]**.
- Open from `agent/&lt;name&gt;` branch/worktree.
