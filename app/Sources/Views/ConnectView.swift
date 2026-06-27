import SwiftUI

// MARK: - Sign-in screen
//
// In Live mode "Connect Spotify" launches ASWebAuthenticationSession (via AppModel.connect).
// In Mock mode it jumps straight in. The real OAuth wiring is owned by spotify-connection.

struct ConnectView: View {
    @Environment(AppModel.self) private var appModel

    var body: some View {
        VStack(spacing: 28) {
            Image(systemName: "music.note.list")
                .font(.system(size: 72, weight: .semibold))
                .foregroundStyle(.tint)
                .padding(28)
                .background(.thinMaterial, in: Circle())

            VStack(spacing: 8) {
                Text("vision-spots")
                    .font(.extraLargeTitle.weight(.bold))
                Text("Your Spotify, in space.")
                    .font(.title3)
                    .foregroundStyle(.secondary)
            }

            Button {
                Task { await appModel.connect() }
            } label: {
                Label("Connect Spotify", systemImage: "link")
                    .font(.title3.weight(.semibold))
                    .padding(.horizontal, 12)
                    .padding(.vertical, 6)
            }
            .buttonStyle(.borderedProminent)
            .controlSize(.extraLarge)

            if case let .failed(message) = appModel.authState {
                Text(message)
                    .font(.callout)
                    .foregroundStyle(.red)
                    .multilineTextAlignment(.center)
                    .frame(maxWidth: 420)
            }

            if !AppConfig.useLiveBackend {
                Label("Running on mock data", systemImage: "ladybug")
                    .font(.caption)
                    .foregroundStyle(.secondary)
            }
        }
        .padding(60)
        .frame(maxWidth: .infinity, maxHeight: .infinity)
    }
}
