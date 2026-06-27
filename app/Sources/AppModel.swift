import Foundation
import Observation

// MARK: - Root app state
//
// Holds the chosen SpotifyService and the auth state the whole UI keys off of. Injected into
// the SwiftUI environment by VisionSpotsApp.

@MainActor
@Observable
final class AppModel {

    enum AuthState: Equatable {
        case signedOut
        case connecting
        case signedIn(UserProfile)
        case failed(String)
    }

    private(set) var authState: AuthState = .signedOut

    let service: any SpotifyService
    private let auth = AuthController()
    private let sessionStore: KeychainStore

    init(service: any SpotifyService = AppConfig.makeService(),
         sessionStore: KeychainStore = .shared) {
        self.service = service
        self.sessionStore = sessionStore
    }

    /// Called at launch to silently restore a prior session from the Keychain.
    func bootstrap() async {
        guard AppConfig.useLiveBackend else { return }
        guard await sessionStore.isSignedIn else { return }
        authState = .connecting
        do {
            let profile = try await service.me()
            authState = .signedIn(profile)
        } catch {
            // Token is stale or backend unreachable — fall back to sign-in screen.
            authState = .signedOut
        }
    }

    /// Connect to Spotify. In Live mode this runs the real OAuth flow; in Mock mode it skips
    /// straight to fetching the (canned) profile so design work needs no backend.
    func connect() async {
        authState = .connecting
        do {
            if AppConfig.useLiveBackend {
                try await auth.signIn()
            }
            let profile = try await service.me()
            authState = .signedIn(profile)
        } catch {
            authState = .failed(error.localizedDescription)
        }
    }

    func signOut() async {
        await auth.signOut()
        authState = .signedOut
    }

    var currentUser: UserProfile? {
        if case let .signedIn(profile) = authState { return profile }
        return nil
    }
}
