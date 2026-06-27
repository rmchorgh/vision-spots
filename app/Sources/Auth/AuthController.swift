import AuthenticationServices
import Foundation

// MARK: - OAuth launcher
//
// Drives the brokered Spotify login from the app side (api-contract.md → "The auth flow"):
//   1. GET {backend}/auth/start  → { authorize_url }
//   2. open authorize_url in ASWebAuthenticationSession
//   3. Spotify → backend /callback → redirects to visionspots://callback?session=<jwt>
//   4. store the jwt in the Keychain
//
// The plumbing is real and ready. In Mock mode the app skips this entirely (see AppModel),
// so design work doesn't need a backend. The `spotify-connection` agent verifies the live
// path end-to-end.

@MainActor
final class AuthController: NSObject {

    private let sessionStore: KeychainStore

    init(sessionStore: KeychainStore = .shared) {
        self.sessionStore = sessionStore
    }

    /// Runs the full web-auth flow and persists the resulting session JWT.
    func signIn() async throws {
        let authorizeURL = try await fetchAuthorizeURL()
        let callbackURL = try await presentWebAuth(url: authorizeURL)
        guard let token = sessionToken(from: callbackURL) else {
            throw SpotifyError.backend("Callback did not contain a session token")
        }
        await sessionStore.setSessionToken(token)
    }

    func signOut() async {
        await sessionStore.setSessionToken(nil)
    }

    // MARK: steps

    private func fetchAuthorizeURL() async throws -> URL {
        let url = AppConfig.backendBaseURL.appendingPathComponent("auth/start")
        let (data, response) = try await URLSession.shared.data(from: url)
        guard (response as? HTTPURLResponse)?.statusCode == 200,
              let json = try? JSONSerialization.jsonObject(with: data) as? [String: Any],
              let urlString = json["authorize_url"] as? String,
              let authorizeURL = URL(string: urlString) else {
            throw SpotifyError.backend("Could not start auth (is the backend reachable?)")
        }
        return authorizeURL
    }

    private func presentWebAuth(url: URL) async throws -> URL {
        try await withCheckedThrowingContinuation { continuation in
            let session = ASWebAuthenticationSession(
                url: url,
                callbackURLScheme: AppConfig.callbackScheme
            ) { callbackURL, error in
                if let callbackURL {
                    continuation.resume(returning: callbackURL)
                } else {
                    continuation.resume(throwing: error ?? SpotifyError.backend("Auth cancelled"))
                }
            }
            session.presentationContextProvider = self
            session.prefersEphemeralWebBrowserSession = false
            session.start()
        }
    }

    private func sessionToken(from url: URL) -> String? {
        URLComponents(url: url, resolvingAgainstBaseURL: false)?
            .queryItems?.first(where: { $0.name == "session" })?.value
    }
}

extension AuthController: ASWebAuthenticationPresentationContextProviding {
    func presentationAnchor(for session: ASWebAuthenticationSession) -> ASPresentationAnchor {
        ASPresentationAnchor()
    }
}
