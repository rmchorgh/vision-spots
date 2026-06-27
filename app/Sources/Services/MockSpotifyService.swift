import Foundation

// MARK: - Canned data for design iteration
//
// Returns believable data with small artificial delays so loading states are visible.
// Owned by the `ui` agent. No network, no auth — `connect()` against this always succeeds.

actor MockSpotifyService: SpotifyService {

    private func delay(_ ms: UInt64 = 350) async {
        try? await Task.sleep(nanoseconds: ms * 1_000_000)
    }

    // Stable image source so the grid looks real (picsum gives deterministic images by seed).
    private func art(_ seed: String, _ size: Int = 400) -> URL? {
        URL(string: "https://picsum.photos/seed/\(seed)/\(size)")
    }

    func me() async throws -> UserProfile {
        await delay()
        return UserProfile(id: "mockuser", displayName: "Richard", product: "premium",
                           imageURL: art("avatar", 200))
    }

    func playlists() async throws -> [Playlist] {
        await delay()
        return [
            Playlist(id: "p1", name: "Deep Focus", description: "Keep calm and focus.",
                     ownerName: "Spotify", trackCount: 152, artworkURL: art("focus")),
            Playlist(id: "p2", name: "Late Night Drive", description: "Synthwave & city lights.",
                     ownerName: "Richard", trackCount: 64, artworkURL: art("drive")),
            Playlist(id: "p3", name: "Coding Beats", description: "Lo-fi for shipping code.",
                     ownerName: "Richard", trackCount: 88, artworkURL: art("coding")),
            Playlist(id: "p4", name: "Morning Coffee", description: "Easy acoustic mornings.",
                     ownerName: "Spotify", trackCount: 120, artworkURL: art("coffee")),
            Playlist(id: "p5", name: "Workout Pump", description: "High energy.",
                     ownerName: "Richard", trackCount: 45, artworkURL: art("workout")),
            Playlist(id: "p6", name: "Rainy Day Jazz", description: "Smooth and mellow.",
                     ownerName: "Spotify", trackCount: 73, artworkURL: art("jazz")),
        ]
    }

    func savedAlbums() async throws -> [Album] {
        await delay()
        let artists = [Artist(id: "a1", name: "Tycho")]
        return [
            Album(id: "al1", name: "Dive", artists: artists, trackCount: 10, artworkURL: art("dive")),
            Album(id: "al2", name: "Awake", artists: artists, trackCount: 8, artworkURL: art("awake")),
            Album(id: "al3", name: "Epoch", artists: [Artist(id: "a2", name: "Bonobo")],
                  trackCount: 12, artworkURL: art("epoch")),
            Album(id: "al4", name: "Migration", artists: [Artist(id: "a2", name: "Bonobo")],
                  trackCount: 12, artworkURL: art("migration")),
        ]
    }

    func recentlyPlayed() async throws -> [Track] {
        await delay()
        return makeTracks(seedPrefix: "recent", count: 8)
    }

    func playlistTracks(id: String) async throws -> [Track] {
        await delay()
        return makeTracks(seedPrefix: "pl-\(id)", count: 18)
    }

    func search(query: String) async throws -> SearchResults {
        await delay(500)
        guard !query.trimmingCharacters(in: .whitespaces).isEmpty else { return SearchResults() }
        return SearchResults(
            tracks: makeTracks(seedPrefix: "q-\(query)", count: 6),
            albums: try await savedAlbums(),
            playlists: Array(try await playlists().prefix(3))
        )
    }

    // MARK: Playback (pretend there's one Connect device)

    private static let mockDevice = Device(id: "d1", name: "Living Room Speaker",
                                           type: "Speaker", isActive: true, volumePercent: 65)

    func devices() async throws -> [Device] {
        await delay(200)
        return [
            Self.mockDevice,
            Device(id: "d2", name: "Richard's iPhone", type: "Smartphone", isActive: false, volumePercent: 80),
            Device(id: "d3", name: "MacBook Pro", type: "Computer", isActive: false, volumePercent: 50),
        ]
    }

    func playbackState() async throws -> PlaybackState {
        await delay(150)
        return PlaybackState(isPlaying: true,
                             track: makeTracks(seedPrefix: "now", count: 1).first,
                             device: Self.mockDevice, progressMs: 42_000)
    }

    // No-ops in mock mode — the views just call them.
    func play(contextURI: String?, deviceID: String?) async throws { await delay(120) }
    func pause() async throws { await delay(120) }
    func next() async throws { await delay(120) }
    func previous() async throws { await delay(120) }
    func transferPlayback(toDeviceID: String) async throws { await delay(120) }

    // MARK: helpers

    private func makeTracks(seedPrefix: String, count: Int) -> [Track] {
        let names = ["Aurora", "Glass", "Horizon", "Drift", "Polaris", "Solstice", "Echoes",
                     "Cascade", " Member", "Lanterns", "Coastline", "Northern", "Slow Burn",
                     "Daydream", "Undertow", "Stillness", "Wanderer", "Afterglow"]
        let artistPool = ["Tycho", "Bonobo", "Boards of Canada", "Emancipator", "ODESZA"]
        return (0..<count).map { i in
            Track(id: "\(seedPrefix)-\(i)",
                  name: names[i % names.count],
                  artists: [Artist(id: "ar\(i)", name: artistPool[i % artistPool.count])],
                  albumName: "Album \(i % 4 + 1)",
                  durationMs: (180 + (i * 17) % 120) * 1000,
                  artworkURL: art("\(seedPrefix)-\(i)"))
        }
    }
}
