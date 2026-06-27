# app — vision-spots (visionOS)

Native visionOS (SwiftUI) client. Bundle id `org.richardmch.visionspots`. The Xcode project
is generated from [`project.yml`](project.yml) with [XcodeGen](https://github.com/yonaskolb/XcodeGen),
so it stays diffable.

## Run it

```bash
brew install xcodegen          # one-time
./regen.sh                     # generate VisionSpots.xcodeproj
xed .                          # open in Xcode → run on the Vision Pro simulator
```

> The first run needs the **visionOS Simulator runtime** (Xcode → Settings → Components, or
> `xcodebuild -downloadPlatform visionOS`). Without it the project still compiles but can't
> launch a simulator.

To type-check from the CLI without a simulator runtime:
```bash
SDK=$(xcrun --sdk xrsimulator --show-sdk-path)
xcrun swiftc -typecheck -sdk "$SDK" -target arm64-apple-xros2.0-simulator -swift-version 6 $(find Sources -name '*.swift')
```

## Mock vs Live data — the one switch

`Sources/Support/Constants.swift` → `AppConfig.useLiveBackend`:

- `false` (default) → **`MockSpotifyService`**: canned data, no backend, no auth. "Connect
  Spotify" jumps straight in. This is how you iterate on UI today.
- `true` → **`LiveSpotifyService`**: real OAuth via `AuthController` + calls to
  `https://vision-spots.richardmch.org`. Flipped on by the `spotify-connection` agent once
  the backend is live.

## Layout

```
Sources/
  VisionSpotsApp.swift     @main; builds AppModel + PlayerModel, injects into environment
  AppModel.swift           auth state + chosen SpotifyService
  PlayerModel.swift        Spotify Connect playback state + transport
  Models/Models.swift      app-side domain types (decoupled from Spotify JSON)
  Services/
    SpotifyService.swift   the protocol every screen talks to  ← the UI↔data seam
    MockSpotifyService.swift
    LiveSpotifyService.swift   skeleton owned by spotify-connection (TODOs inside)
  Auth/
    AuthController.swift   ASWebAuthenticationSession OAuth flow
    KeychainStore.swift    stores the vision-spots session JWT
  Views/
    RootView / ConnectView / MainView
    LibraryView / PlaylistDetailView / SearchView
    NowPlayingBar (ornament) / DevicePickerView
    Components/ ArtworkView, TrackRow
  Support/Constants.swift  AppConfig (the Mock↔Live switch, backend URL, scheme)
Resources/
  Info.plist               CFBundleURLTypes registers the visionspots:// scheme
  Assets.xcassets          AccentColor + placeholder layered AppIcon
Signing.xcconfig           placeholder; apple-signing fills DEVELOPMENT_TEAM
```

## Handoff notes for other agents
- **`apple-signing`**: set `DEVELOPMENT_TEAM` in `Signing.xcconfig`; bundle id is already wired.
- **`spotify-connection`**: implement `LiveSpotifyService` (maps `docs/contracts/api-contract.md`),
  then flip `AppConfig.useLiveBackend = true`. `AuthController` + `KeychainStore` are ready.
