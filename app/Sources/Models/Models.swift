import Foundation

// MARK: - Domain models
//
// These are the app's own view models — deliberately decoupled from Spotify's raw JSON.
// The `spotify-connection` agent maps backend responses (see docs/contracts/api-contract.md)
// into these types inside `LiveSpotifyService`, so the views never change when data goes live.

struct UserProfile: Identifiable, Hashable {
    let id: String
    let displayName: String
    /// "premium" or "free". Spotify Connect playback control requires Premium.
    let product: String
    let imageURL: URL?

    var isPremium: Bool { product.lowercased() == "premium" }
}

struct Artist: Identifiable, Hashable {
    let id: String
    let name: String
}

struct Track: Identifiable, Hashable {
    let id: String
    let name: String
    let artists: [Artist]
    let albumName: String
    let durationMs: Int
    let artworkURL: URL?

    var uri: String { "spotify:track:\(id)" }
    var artistNames: String { artists.map(\.name).joined(separator: ", ") }

    /// "m:ss" for track rows.
    var durationFormatted: String {
        let totalSeconds = durationMs / 1000
        return String(format: "%d:%02d", totalSeconds / 60, totalSeconds % 60)
    }
}

struct Playlist: Identifiable, Hashable {
    let id: String
    let name: String
    let description: String
    let ownerName: String
    let trackCount: Int
    let artworkURL: URL?

    var uri: String { "spotify:playlist:\(id)" }
}

struct Album: Identifiable, Hashable {
    let id: String
    let name: String
    let artists: [Artist]
    let trackCount: Int
    let artworkURL: URL?

    var uri: String { "spotify:album:\(id)" }
    var artistNames: String { artists.map(\.name).joined(separator: ", ") }
}

/// A Spotify Connect playback target (phone, speaker, computer, …).
struct Device: Identifiable, Hashable {
    let id: String
    let name: String
    let type: String
    let isActive: Bool
    let volumePercent: Int?
}

/// Snapshot of what's playing on the active Connect device.
struct PlaybackState: Hashable {
    var isPlaying: Bool
    var track: Track?
    var device: Device?
    var progressMs: Int

    static let idle = PlaybackState(isPlaying: false, track: nil, device: nil, progressMs: 0)
}

struct SearchResults: Hashable {
    var tracks: [Track] = []
    var albums: [Album] = []
    var playlists: [Playlist] = []

    var isEmpty: Bool { tracks.isEmpty && albums.isEmpty && playlists.isEmpty }
}
