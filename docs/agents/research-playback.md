# Agent: `research-playback`

**Phase A.** Pure research. Owns `docs/`. Blocks nobody. Goal: determine whether we can
replicate **iPad-style in-app Spotify playback** on Apple Vision Pro, instead of only
remote-controlling another device via Spotify Connect.

## Why this exists
For v1, playback works through **Spotify Connect** (control audio on a phone/speaker/
computer). The owner wants to know if real **in-app** audio playback is achievable on
Vision Pro the way it works on iPad. Find out, with evidence, and recommend a path.

## Read first
- [`../../ORCHESTRATION.md`](../../ORCHESTRATION.md)
- [`../contracts/shared-constants.md`](../contracts/shared-constants.md)

## Questions to answer (with sources/links)
1. **iOS SDK on visionOS.** Does the Spotify iOS SDK (`SpotifyiOS.framework` / App Remote)
   run on visionOS, in the "Designed for iPad" compatibility mode, or via Mac Catalyst?
   What exactly does App Remote do — does it stream audio in-app, or does it *also* just
   remote-control the Spotify app (which doesn't exist on visionOS)?
2. **"Designed for iPad" path.** Can the iPad Spotify app (or our own iPad build using the
   SDK) run unmodified on Vision Pro? What breaks? Does audio play?
3. **Web Playback SDK.** Spotify's Web Playback SDK streams audio in a browser via EME/DRM
   (Widevine). Does it work in a `WKWebView` on visionOS? Test or find evidence. Premium-only.
4. **Licensing / ToS.** Does Spotify's Developer Terms permit a third-party app to stream
   full audio? (Historically full playback is restricted to first-party + certified
   partners; Web Playback SDK is the sanctioned in-browser path, Premium-only.) Cite the
   relevant terms so we don't build something that gets the app key revoked.
5. **Audio session / spatial.** If in-app playback is viable, how does audio routing and
   AVAudioSession behave on visionOS? Any spatial-audio considerations?

## How to work
- Use web search + official Spotify developer docs + Apple visionOS docs. Cite every claim
  with a link. Where you can, **prototype**: a tiny throwaway visionOS test (WKWebView +
  Web Playback SDK token, or a "Designed for iPad" build with App Remote) to confirm
  behavior rather than guessing. Put any throwaway prototype under
  `docs/research/prototypes/` clearly marked as throwaway.
- Don't modify `app/` or `backend/`. Your deliverable is a decision document.

## Deliverable
`docs/research/playback-options.md` containing:
- A comparison table: **Spotify Connect (v1)** vs **Web Playback SDK in WebView** vs
  **iPad-compat App Remote** — feasibility, Premium requirement, ToS risk, audio-in-app
  (yes/no), spatial audio, effort.
- A clear **recommendation** and a proposed follow-up task if in-app playback is viable.
- Any prototype results / screenshots / console output that back the conclusion.

## Definition of done
- `docs/research/playback-options.md` written with sourced claims and a recommendation.
- PR opened from `agent/research-playback`. No production code changed.
