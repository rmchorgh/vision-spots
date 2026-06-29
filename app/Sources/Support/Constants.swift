import Foundation
import SwiftUI

// MARK: - App configuration
//
// Values here come from docs/contracts/shared-constants.md. Keep them in sync.

enum AppConfig {

    /// THE switch the `ui` agent documents: which `SpotifyService` the app uses.
    /// Defaults to Mock so design/dev needs no backend. Build with `-D USE_LIVE_BACKEND`
    /// (set `USE_LIVE_BACKEND` in the scheme's "Other Swift Flags" / a `Live` config) to
    /// run against the real backend + Spotify — no source edit required for the swap.
    static let useLiveBackend: Bool = {
        #if USE_LIVE_BACKEND
        return true
        #else
        return false
        #endif
    }()

    /// Self-hosted Go auth/token service.
    static let backendBaseURL = URL(string: "https://vision-spots.richardmch.org")!

    /// Deep-link scheme Spotify's callback redirects back into (visionspots://callback).
    static let callbackScheme = "visionspots"

    /// Factory used by the app entry point. Single source for the Mock ↔ Live decision.
    static func makeService() -> any SpotifyService {
        useLiveBackend ? LiveSpotifyService() : MockSpotifyService()
    }
}

// MARK: - Look to Scroll (visionOS 26)
//
// Gaze-driven scrolling. The project targets visionOS 2.0, so the modifier is guarded and
// compiles to a no-op on earlier systems.

extension View {
    @ViewBuilder
    func lookToScroll() -> some View {
        if #available(visionOS 26.0, *) {
            scrollInputBehavior(.enabled, for: .look)
        } else {
            self
        }
    }
}
