import Foundation

// MARK: - Real backend implementation
//
// Owned by the `spotify-connection` agent (docs/agents/spotify-connection.md).
// Maps the backend responses described in docs/contracts/api-contract.md into the app's
// domain models (Models.swift). Browse/search go through the generic `/api/spotify/*`
// proxy; playback transport uses the typed `/api/player/*` endpoints. Volume and device
// transfer have no typed route yet, so they ride the generic proxy too.
//
// Mock → Live is flipped via `AppConfig.useLiveBackend` (a build flag, see Constants.swift).

actor LiveSpotifyService: SpotifyService {

    private let baseURL = AppConfig.backendBaseURL
    private let session: URLSession
    private let sessionStore: KeychainStore
    private let decoder = JSONDecoder()

    init(sessionStore: KeychainStore = .shared) {
        self.session = URLSession(configuration: .default)
        self.sessionStore = sessionStore
    }

    // MARK: Authenticated request helper

    /// Performs a backend request with the stored vision-spots session JWT, and transparently
    /// refreshes once on `session_expired` (per api-contract.md). Returns the raw 2xx body
    /// (which may be empty, e.g. a 204 from a player transport call).
    ///
    /// `path` may include a query string — we build the URL by string concatenation rather
    /// than `appendingPathComponent`, which would percent-encode the `?` and corrupt it.
    private func authedData(for path: String,
                            method: String = "GET",
                            body: Data? = nil) async throws -> Data {
        guard let token = await sessionStore.sessionToken else { throw SpotifyError.sessionExpired }
        guard let endpoint = URL(string: baseURL.absoluteString + "/" + path) else {
            throw SpotifyError.backend("Invalid request path: \(path)")
        }

        func makeRequest(_ token: String) -> URLRequest {
            var req = URLRequest(url: endpoint)
            req.httpMethod = method
            req.setValue("Bearer \(token)", forHTTPHeaderField: "Authorization")
            if let body { req.httpBody = body; req.setValue("application/json", forHTTPHeaderField: "Content-Type") }
            return req
        }

        var (data, response) = try await session.data(for: makeRequest(token))

        // One silent refresh attempt on 401, then retry.
        if (response as? HTTPURLResponse)?.statusCode == 401 {
            let refreshed = try await refreshSession()
            (data, response) = try await session.data(for: makeRequest(refreshed))
        }

        guard let http = response as? HTTPURLResponse else { throw SpotifyError.backend("No HTTP response") }
        switch http.statusCode {
        case 200..<300: return data
        case 401:       throw SpotifyError.sessionExpired
        case 403:       throw SpotifyError.premiumRequired   // backend maps premium_required → 403
        case 404:       throw SpotifyError.noActiveDevice
        case 429:       throw SpotifyError.backend("Spotify rate limit reached. Try again in a moment.")
        default:        throw SpotifyError.backend(backendMessage(from: data) ?? "Backend returned \(http.statusCode)")
        }
    }

    /// Calls POST /auth/refresh and stores the new session token.
    @discardableResult
    private func refreshSession() async throws -> String {
        guard let token = await sessionStore.sessionToken else { throw SpotifyError.sessionExpired }
        var req = URLRequest(url: baseURL.appendingPathComponent("auth/refresh"))
        req.httpMethod = "POST"
        req.setValue("Bearer \(token)", forHTTPHeaderField: "Authorization")
        let (data, response) = try await session.data(for: req)
        guard (response as? HTTPURLResponse)?.statusCode == 200,
              let json = try? JSONSerialization.jsonObject(with: data) as? [String: Any],
              let newToken = json["session"] as? String else {
            throw SpotifyError.sessionExpired
        }
        await sessionStore.setSessionToken(newToken)
        return newToken
    }

    // MARK: Decoding helpers

    /// GETs `path` and decodes the JSON body as `T`.
    private func get<T: Decodable>(_ path: String) async throws -> T {
        let data = try await authedData(for: path)
        guard !data.isEmpty else { throw SpotifyError.backend("Empty response from \(path)") }
        do {
            return try decoder.decode(T.self, from: data)
        } catch {
            throw SpotifyError.backend("Could not read Spotify response (\(path)).")
        }
    }

    /// Extracts the backend's human-readable `message` from an error envelope, if present.
    private func backendMessage(from data: Data) -> String? {
        guard let json = try? JSONSerialization.jsonObject(with: data) as? [String: Any] else { return nil }
        return json["message"] as? String
    }

    // MARK: SpotifyService — identity

    func me() async throws -> UserProfile {
        let dto: MeDTO = try await get("me")
        return UserProfile(
            id: dto.id,
            displayName: dto.display_name?.isEmpty == false ? dto.display_name! : dto.id,
            product: dto.product ?? "free",
            imageURL: dto.image.flatMap { $0.isEmpty ? nil : URL(string: $0) })
    }

    // MARK: SpotifyService — library / browse

    func playlists() async throws -> [Playlist] {
        let page: Paging<PlaylistDTO> = try await get("api/spotify/me/playlists?limit=50")
        return (page.items ?? []).compactMap(mapPlaylist)
    }

    func savedAlbums() async throws -> [Album] {
        let page: Paging<SavedAlbumItem> = try await get("api/spotify/me/albums?limit=50")
        return (page.items ?? []).compactMap { $0.album.flatMap(mapAlbum) }
    }

    func playlistTracks(id: String, offset: Int, limit: Int) async throws -> TrackPage {
        if id == Playlist.likedSongsID {
            // Liked Songs has no typed endpoint; page the generic /me/tracks proxy.
            let page: Paging<PlaylistTrackItem> = try await get("api/spotify/me/tracks?limit=\(limit)&offset=\(offset)")
            let tracks = (page.items ?? []).compactMap { $0.track.flatMap(mapTrack) }
            let total = page.total ?? (offset + tracks.count)
            let next = offset + tracks.count < total ? offset + tracks.count : nil
            return TrackPage(tracks: tracks, total: total, nextOffset: next)
        }

        // Real playlists use the typed, app-shaped endpoint (one combined artist string,
        // track_id is a `spotify:track:…` URI). next_offset comes straight from the backend.
        let page: TypedTrackPageDTO = try await get("api/playlists/\(id)/tracks?limit=\(limit)&offset=\(offset)")
        return TrackPage(
            tracks: (page.items ?? []).compactMap(mapTypedTrack),
            total: page.total ?? 0,
            nextOffset: page.next_offset)
    }

    func albumTracks(id: String, offset: Int, limit: Int) async throws -> TrackPage {
        // Generic proxy: album-track objects omit album art, so the detail view supplies a
        // fallback. We don't get track-level images here.
        let page: Paging<TrackDTO> = try await get("api/spotify/albums/\(id)/tracks?limit=\(limit)&offset=\(offset)")
        let tracks = (page.items ?? []).compactMap(mapTrack)
        let total = page.total ?? (offset + tracks.count)
        let next = offset + tracks.count < total ? offset + tracks.count : nil
        return TrackPage(tracks: tracks, total: total, nextOffset: next)
    }

    // MARK: SpotifyService — home

    func daylist() async throws -> Playlist? {
        // Spotify exposes no public Web API endpoint for the algorithmic "daylist", and it
        // isn't reliably identifiable among `/me/playlists`. Surface nothing rather than
        // guessing; HomeView already renders gracefully when this is nil.
        nil
    }

    func likedSongs() async throws -> Playlist {
        // Only need the total here; one item keeps the payload tiny.
        let page: Paging<PlaylistTrackItem> = try await get("api/spotify/me/tracks?limit=1")
        return Playlist(
            id: Playlist.likedSongsID,
            name: "Liked Songs",
            description: "Every song you've liked, in one place.",
            ownerName: "You",
            trackCount: page.total ?? 0,
            artworkURL: nil)   // rendered with the signature purple gradient instead of art
    }

    func recentlyPlayed() async throws -> [MediaItem] {
        let page: Paging<RecentlyPlayedItem> = try await get("api/spotify/me/player/recently-played?limit=50")

        var items: [MediaItem] = []
        var seen = Set<String>()
        var playlistIDs: [String] = []

        for entry in page.items ?? [] {
            if entry.context?.type == "playlist",
               let uri = entry.context?.uri,
               let pid = uri.split(separator: ":").last.map(String.init) {
                // Defer playlist fetches: they each need a metadata round-trip.
                if seen.insert("playlist:\(pid)").inserted, playlistIDs.count < 6 {
                    playlistIDs.append(pid)
                }
            } else if let albumDTO = entry.track?.album, let album = mapAlbum(albumDTO) {
                if seen.insert("album:\(album.id)").inserted {
                    items.append(.album(album))
                }
            }
        }

        // Resolve the playlist contexts (bounded above to 6). A failed fetch just drops that one.
        for pid in playlistIDs {
            if let p = try? await playlistMeta(id: pid) {
                items.append(.playlist(p))
            }
        }

        return Array(items.prefix(18))
    }

    private func playlistMeta(id: String) async throws -> Playlist {
        let dto: PlaylistDTO = try await get(
            "api/spotify/playlists/\(id)?fields=id,name,description,owner(display_name),tracks(total),images")
        guard let p = mapPlaylist(dto) else { throw SpotifyError.backend("Malformed playlist \(id)") }
        return p
    }

    // MARK: SpotifyService — search

    func search(query: String) async throws -> SearchResults {
        let trimmed = query.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmed.isEmpty else { return SearchResults() }
        let q = trimmed.addingPercentEncoding(withAllowedCharacters: .urlQueryAllowed) ?? trimmed
        let dto: SearchDTO = try await get("api/spotify/search?q=\(q)&type=track,album,playlist&limit=20")
        return SearchResults(
            tracks: (dto.tracks?.items ?? []).compactMap(mapTrack),
            albums: (dto.albums?.items ?? []).compactMap(mapAlbum),
            playlists: (dto.playlists?.items ?? []).compactMap(mapPlaylist))
    }

    // MARK: SpotifyService — playback control (Spotify Connect)

    func devices() async throws -> [Device] {
        let dto: DevicesDTO = try await get("api/player/devices")
        return (dto.devices ?? []).compactMap(mapDevice)
    }

    func playbackState() async throws -> PlaybackState {
        // /api/player/state returns 204 (empty body) when nothing is playing.
        let data = try await authedData(for: "api/player/state")
        guard !data.isEmpty, let dto = try? decoder.decode(PlaybackStateDTO.self, from: data) else {
            return .idle
        }
        return PlaybackState(
            isPlaying: dto.is_playing ?? false,
            track: dto.item.flatMap(mapTrack),
            device: dto.device.flatMap(mapDevice),
            progressMs: dto.progress_ms ?? 0)
    }

    func play(contextURI: String?, deviceID: String?) async throws {
        var payload: [String: Any] = [:]
        if let deviceID { payload["device_id"] = deviceID }

        if let uri = contextURI {
            let likedURI = "spotify:playlist:\(Playlist.likedSongsID)"
            let daylistURI = "spotify:playlist:\(Playlist.daylistID)"
            if uri == likedURI {
                // "Liked Songs" isn't a real playlist context — play its tracks as a queue.
                let uris = try await likedTrackURIs(limit: 50)
                if !uris.isEmpty { payload["uris"] = uris }
            } else if uri == daylistURI {
                // Synthetic id with no real context; fall through to "resume".
            } else if uri.hasPrefix("spotify:track:") {
                // A single track must be passed as `uris`, not `context_uri`.
                payload["uris"] = [uri]
            } else {
                payload["context_uri"] = uri
            }
        }

        let body = payload.isEmpty ? nil : try JSONSerialization.data(withJSONObject: payload)
        _ = try await authedData(for: "api/player/play", method: "PUT", body: body)
    }

    func pause() async throws {
        _ = try await authedData(for: "api/player/pause", method: "PUT")
    }

    func next() async throws {
        _ = try await authedData(for: "api/player/next", method: "POST")
    }

    func previous() async throws {
        _ = try await authedData(for: "api/player/previous", method: "POST")
    }

    func transferPlayback(toDeviceID: String) async throws {
        // No typed backend route; go through the generic proxy to Spotify's PUT /me/player.
        let body = try JSONSerialization.data(withJSONObject: ["device_ids": [toDeviceID], "play": true])
        _ = try await authedData(for: "api/spotify/me/player", method: "PUT", body: body)
    }

    func setVolume(percent: Int) async throws {
        let clamped = min(100, max(0, percent))
        _ = try await authedData(for: "api/spotify/me/player/volume?volume_percent=\(clamped)", method: "PUT")
    }

    func seek(toPositionMs ms: Int) async throws {
        _ = try await authedData(for: "api/spotify/me/player/seek?position_ms=\(max(0, ms))", method: "PUT")
    }

    // MARK: Private helpers

    private func likedTrackURIs(limit: Int) async throws -> [String] {
        let page: Paging<PlaylistTrackItem> = try await get("api/spotify/me/tracks?limit=\(limit)")
        return (page.items ?? []).compactMap { $0.track?.id }.map { "spotify:track:\($0)" }
    }

    // MARK: DTO → model mapping

    private func mapArtists(_ artists: [ArtistDTO]?) -> [Artist] {
        (artists ?? []).compactMap { dto in
            guard let id = dto.id, let name = dto.name else { return nil }
            return Artist(id: id, name: name)
        }
    }

    private func firstImageURL(_ images: [ImageDTO]?) -> URL? {
        images?.compactMap { $0.url }.first(where: { !$0.isEmpty }).flatMap(URL.init)
    }

    private func mapTrack(_ d: TrackDTO) -> Track? {
        guard let id = d.id else { return nil }   // local/unavailable tracks have no id
        return Track(
            id: id,
            name: d.name ?? "",
            artists: mapArtists(d.artists),
            albumName: d.album?.name ?? "",
            durationMs: d.duration_ms ?? 0,
            artworkURL: firstImageURL(d.album?.images))
    }

    /// Maps the typed playlist-track item (single combined artist string, `spotify:track:…`
    /// URI) into a `Track`. The bare id is recovered by stripping the URI prefix so
    /// `Track.uri` round-trips correctly.
    private func mapTypedTrack(_ d: TypedTrackItem) -> Track? {
        guard let uri = d.track_id else { return nil }
        let id = uri.split(separator: ":").last.map(String.init) ?? uri
        let artists = (d.artist?.isEmpty == false) ? [Artist(id: "\(id)-artist", name: d.artist!)] : []
        return Track(
            id: id,
            name: d.name ?? "",
            artists: artists,
            albumName: d.album ?? "",
            durationMs: d.duration_ms ?? 0,
            artworkURL: d.image.flatMap { $0.isEmpty ? nil : URL(string: $0) })
    }

    private func mapAlbum(_ d: AlbumDTO) -> Album? {
        guard let id = d.id else { return nil }
        return Album(
            id: id,
            name: d.name ?? "",
            artists: mapArtists(d.artists),
            trackCount: d.total_tracks ?? 0,
            artworkURL: firstImageURL(d.images))
    }

    private func mapPlaylist(_ d: PlaylistDTO) -> Playlist? {
        guard let id = d.id else { return nil }
        return Playlist(
            id: id,
            name: d.name ?? "",
            description: d.description ?? "",
            ownerName: d.owner?.display_name ?? "",
            trackCount: d.tracks?.total ?? 0,
            artworkURL: firstImageURL(d.images))
    }

    private func mapDevice(_ d: DeviceDTO) -> Device? {
        guard let id = d.id else { return nil }
        return Device(
            id: id,
            name: d.name ?? "Unknown device",
            type: d.type ?? "Unknown",
            isActive: d.is_active ?? false,
            volumePercent: d.volume_percent)
    }
}

// MARK: - Spotify Web API DTOs
//
// Minimal, lenient shapes — everything optional so a missing field never fails the whole
// decode. Property names match Spotify's snake_case JSON so no CodingKeys are needed.

private struct MeDTO: Decodable {
    let id: String
    let display_name: String?
    let product: String?
    let image: String?
}

private struct ImageDTO: Decodable { let url: String? }
private struct ArtistDTO: Decodable { let id: String?; let name: String? }
private struct OwnerDTO: Decodable { let display_name: String? }
private struct TracksRefDTO: Decodable { let total: Int? }

private struct PlaylistDTO: Decodable {
    let id: String?
    let name: String?
    let description: String?
    let owner: OwnerDTO?
    let tracks: TracksRefDTO?
    let images: [ImageDTO]?
}

private struct AlbumDTO: Decodable {
    let id: String?
    let name: String?
    let artists: [ArtistDTO]?
    let total_tracks: Int?
    let images: [ImageDTO]?
}

private struct TrackDTO: Decodable {
    let id: String?
    let name: String?
    let artists: [ArtistDTO]?
    let album: AlbumDTO?
    let duration_ms: Int?
}

private struct Paging<T: Decodable>: Decodable {
    let items: [T]?
    let total: Int?
    let next: String?

    private enum CodingKeys: String, CodingKey { case items, total, next }

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        total = try c.decodeIfPresent(Int.self, forKey: .total)
        next = try c.decodeIfPresent(String.self, forKey: .next)
        // Spotify's /v1/search returns `null` (and occasionally malformed) entries inside
        // `items` — most often in playlists.items. Decode leniently so one bad element
        // doesn't make the whole page (and thus the whole search) throw and come back empty.
        items = try c.decodeIfPresent([FailableDecodable<T>].self, forKey: .items)?.compactMap(\.value)
    }
}

/// Decodes `T` if possible, otherwise yields `nil` instead of throwing — lets an array
/// tolerate null or malformed elements without discarding the entire array.
private struct FailableDecodable<T: Decodable>: Decodable {
    let value: T?
    init(from decoder: Decoder) throws { value = try? T(from: decoder) }
}

private struct SavedAlbumItem: Decodable { let album: AlbumDTO? }
private struct PlaylistTrackItem: Decodable { let track: TrackDTO? }

// The typed, app-shaped playlist-tracks endpoint (`GET /api/playlists/:id/tracks`).
private struct TypedTrackItem: Decodable {
    let track_id: String?
    let name: String?
    let artist: String?
    let album: String?
    let duration_ms: Int?
    let image: String?
}
private struct TypedTrackPageDTO: Decodable {
    let items: [TypedTrackItem]?
    let total: Int?
    let next_offset: Int?
}

private struct DeviceDTO: Decodable {
    let id: String?
    let name: String?
    let type: String?
    let is_active: Bool?
    let volume_percent: Int?
}
private struct DevicesDTO: Decodable { let devices: [DeviceDTO]? }

private struct PlaybackStateDTO: Decodable {
    let is_playing: Bool?
    let progress_ms: Int?
    let device: DeviceDTO?
    let item: TrackDTO?
}

private struct SearchDTO: Decodable {
    let tracks: Paging<TrackDTO>?
    let albums: Paging<AlbumDTO>?
    let playlists: Paging<PlaylistDTO>?
}

private struct ContextDTO: Decodable { let uri: String?; let type: String? }
private struct RecentlyPlayedItem: Decodable {
    let track: TrackDTO?
    let context: ContextDTO?
}
