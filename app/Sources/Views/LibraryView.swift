import SwiftUI

// MARK: - Library: a lazy-loaded view of the Liked Songs playlist
//
// The Library tab is simply the user's Liked Songs, fetched on first appearance and rendered
// with the shared playlist detail view.

struct LibraryView: View {
    @Environment(AppModel.self) private var appModel

    @State private var liked: Playlist?
    @State private var loadState: LoadState = .loading

    var body: some View {
        Group {
            switch loadState {
            case .loading:
                ProgressView().controlSize(.large).frame(maxWidth: .infinity, maxHeight: .infinity)
            case .failed(let message):
                ContentUnavailableView("Couldn't load Library", systemImage: "exclamationmark.triangle",
                                       description: Text(message))
            case .loaded:
                if let liked {
                    PlaylistDetailView(playlist: liked)
                }
            }
        }
        .navigationTitle("Library")
        .task { await load() }
    }

    private func load() async {
        loadState = .loading
        do {
            liked = try await appModel.service.likedSongs()
            loadState = .loaded
        } catch {
            loadState = .failed((error as? SpotifyError)?.errorDescription ?? error.localizedDescription)
        }
    }
}

// Shared loading state for the data-backed screens.
enum LoadState: Equatable {
    case loading
    case loaded
    case failed(String)
}
