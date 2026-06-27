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

    init(service: any SpotifyService = AppConfig.makeService()) {
        self.service = service
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
