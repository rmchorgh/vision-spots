import Foundation

// MARK: - App configuration
//
// Values here come from docs/contracts/shared-constants.md. Keep them in sync.

enum AppConfig {

    /// THE switch the `ui` agent documents: flip to `true` once the `spotify-connection`
    /// agent + a live backend are ready. `false` runs entirely on MockSpotifyService.
    static let useLiveBackend = false

    /// Self-hosted Go auth/token service.
    static let backendBaseURL = URL(string: "https://vision-spots.richardmch.org")!

    /// Deep-link scheme Spotify's callback redirects back into (visionspots://callback).
    static let callbackScheme = "visionspots"

    /// Factory used by the app entry point. Single source for the Mock ↔ Live decision.
    static func makeService() -> any SpotifyService {
        useLiveBackend ? LiveSpotifyService() : MockSpotifyService()
    }
}
