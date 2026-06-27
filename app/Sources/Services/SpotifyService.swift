import Foundation

// MARK: - The seam between UI and data
//
// Every screen talks to this protocol, never to the network directly. The `ui` agent ships
// `MockSpotifyService` (canned data) so design can iterate without a backend; the
// `spotify-connection` agent ships `LiveSpotifyService` (real calls to
// https://vision-spots.richardmch.org). Flipping which one the app uses is one line in
// `AppConfig.useLiveBackend` (see Support/Constants.swift).

protocol SpotifyService: Sendable {
    // Identity
    func me() async throws -> UserProfile

    // Library / browse
    func playlists() async throws -> [Playlist]
    func savedAlbums() async throws -> [Album]
    func recentlyPlayed() async throws -> [Track]
    func playlistTracks(id: String) async throws -> [Track]

    // Search
    func search(query: String) async throws -> SearchResults

    // Playback control via Spotify Connect (maps to backend /api/player/*)
    func devices() async throws -> [Device]
    func playbackState() async throws -> PlaybackState
    func play(contextURI: String?, deviceID: String?) async throws
    func pause() async throws
    func next() async throws
    func previous() async throws
    func transferPlayback(toDeviceID: String) async throws
}

/// Errors surfaced to the UI. Mirrors the backend error shapes in api-contract.md.
enum SpotifyError: LocalizedError {
    case sessionExpired
    case premiumRequired
    case noActiveDevice
    case notImplemented
    case backend(String)

    var errorDescription: String? {
        switch self {
        case .sessionExpired:  return "Your Spotify session expired. Please reconnect."
        case .premiumRequired: return "Spotify Premium is required to control playback."
        case .noActiveDevice:  return "No active Spotify Connect device. Open Spotify on a phone, speaker, or computer."
        case .notImplemented:  return "Not implemented yet — wired up by the spotify-connection agent."
        case .backend(let m):  return m
        }
    }
}
