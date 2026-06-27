import SwiftUI

// MARK: - Top-level switch: Connect screen vs the main app

struct RootView: View {
    @Environment(AppModel.self) private var appModel

    var body: some View {
        Group {
            switch appModel.authState {
            case .signedOut, .failed:
                ConnectView()
            case .connecting:
                ProgressView("Connecting to Spotify…")
                    .controlSize(.large)
            case .signedIn:
                MainView()
            }
        }
        .task { await appModel.bootstrap() }
    }
}
