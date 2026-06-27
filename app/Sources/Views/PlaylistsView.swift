import SwiftUI

// MARK: - Playlists: a grid of the user's playlists

struct PlaylistsView: View {
    @Environment(AppModel.self) private var appModel

    @State private var playlists: [Playlist] = []
    @State private var loadState: LoadState = .loading

    private let columns = [GridItem(.adaptive(minimum: 180, maximum: 220), spacing: 24)]

    var body: some View {
        ScrollView {
            switch loadState {
            case .loading:
                ProgressView().controlSize(.large).frame(maxWidth: .infinity, minHeight: 300)
            case .failed(let message):
                ContentUnavailableView("Couldn't load playlists", systemImage: "exclamationmark.triangle",
                                       description: Text(message))
                    .frame(minHeight: 300)
            case .loaded:
                LazyVGrid(columns: columns, spacing: 24) {
                    ForEach(playlists) { MediaCard(item: .playlist($0)) }
                }
                .padding(28)
            }
        }
        .navigationTitle("Playlists")
        .navigationBarTitleDisplayMode(.inline)
        .task { await load() }
    }

    private func load() async {
        loadState = .loading
        do {
            playlists = try await appModel.service.playlists()
            loadState = .loaded
        } catch {
            loadState = .failed((error as? SpotifyError)?.errorDescription ?? error.localizedDescription)
        }
    }
}
