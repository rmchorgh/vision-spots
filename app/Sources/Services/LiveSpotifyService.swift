import Foundation

// MARK: - Real backend implementation (SKELETON)
//
// ⚠️ Owned by the `spotify-connection` agent (docs/agents/spotify-connection.md).
// The `ui` agent provides this skeleton so the Mock → Live swap is one line. The plumbing
// below (base URL, bearer session header, refresh-on-401) is real and ready; the data
// methods throw `.notImplemented` until spotify-connection maps the JSON described in
// docs/contracts/api-contract.md.

actor LiveSpotifyService: SpotifyService {

    private let baseURL = AppConfig.backendBaseURL
    private let session: URLSession
    private let sessionStore: KeychainStore

    init(sessionStore: KeychainStore = .shared) {
        self.session = URLSession(configuration: .default)
        self.sessionStore = sessionStore
    }

    // MARK: Authenticated request helper (ready to use)

    /// Performs a backend request with the stored vision-spots session JWT, and transparently
    /// refreshes once on `session_expired` (per api-contract.md). spotify-connection builds
    /// typed calls on top of this.
    private func authedData(for path: String,
                            method: String = "GET",
                            body: Data? = nil) async throws -> Data {
        guard let token = await sessionStore.sessionToken else { throw SpotifyError.sessionExpired }

        func makeRequest(_ token: String) -> URLRequest {
            var req = URLRequest(url: baseURL.appendingPathComponent(path))
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
        default:        throw SpotifyError.backend("Backend returned \(http.statusCode)")
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

    // MARK: SpotifyService — TODO(spotify-connection): map api-contract.md responses.

    func me() async throws -> UserProfile { throw SpotifyError.notImplemented }
    func playlists() async throws -> [Playlist] { throw SpotifyError.notImplemented }
    func savedAlbums() async throws -> [Album] { throw SpotifyError.notImplemented }
    func recentlyPlayed() async throws -> [Track] { throw SpotifyError.notImplemented }
    func playlistTracks(id: String) async throws -> [Track] { throw SpotifyError.notImplemented }
    func search(query: String) async throws -> SearchResults { throw SpotifyError.notImplemented }
    func devices() async throws -> [Device] { throw SpotifyError.notImplemented }
    func playbackState() async throws -> PlaybackState { throw SpotifyError.notImplemented }
    func play(contextURI: String?, deviceID: String?) async throws { throw SpotifyError.notImplemented }
    func pause() async throws { throw SpotifyError.notImplemented }
    func next() async throws { throw SpotifyError.notImplemented }
    func previous() async throws { throw SpotifyError.notImplemented }
    func transferPlayback(toDeviceID: String) async throws { throw SpotifyError.notImplemented }
}
