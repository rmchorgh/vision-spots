import SwiftUI

// MARK: - Library: playlists, saved albums, recently played

struct LibraryView: View {
    @Environment(AppModel.self) private var appModel

    @State private var playlists: [Playlist] = []
    @State private var albums: [Album] = []
    @State private var recent: [Track] = []
    @State private var loadState: LoadState = .loading

    private let columns = [GridItem(.adaptive(minimum: 180, maximum: 220), spacing: 20)]

    var body: some View {
        ScrollView {
            switch loadState {
            case .loading:
                ProgressView().controlSize(.large).frame(maxWidth: .infinity, minHeight: 300)
            case .failed(let message):
                ContentUnavailableView("Couldn't load library", systemImage: "exclamationmark.triangle", description: Text(message))
                    .frame(minHeight: 300)
            case .loaded:
                content
            }
        }
        .navigationTitle("Library")
        .task { await load() }
    }

    private var content: some View {
        VStack(alignment: .leading, spacing: 32) {
            sectionHeader("Playlists")
            LazyVGrid(columns: columns, spacing: 20) {
                ForEach(playlists) { playlist in
                    NavigationLink(value: playlist) {
                        CardView(title: playlist.name, subtitle: "\(playlist.trackCount) tracks", artworkURL: playlist.artworkURL)
                    }
                    .buttonStyle(.plain)
                }
            }

            sectionHeader("Saved Albums")
            LazyVGrid(columns: columns, spacing: 20) {
                ForEach(albums) { album in
                    CardView(title: album.name, subtitle: album.artistNames, artworkURL: album.artworkURL)
                }
            }

            sectionHeader("Recently Played")
            VStack(spacing: 0) {
                ForEach(recent) { track in
                    TrackRow(track: track)
                    // Non-conditional separator (avoids _ConditionalContent inside a ForEach).
                    Divider().opacity(track.id == recent.last?.id ? 0 : 1)
                }
            }
            .padding(16)
            .background(.thinMaterial, in: RoundedRectangle(cornerRadius: 16, style: .continuous))
        }
        .padding(28)
    }

    private func sectionHeader(_ title: String) -> some View {
        Text(title).font(.title.weight(.bold))
    }

    private func load() async {
        loadState = .loading
        do {
            async let p = appModel.service.playlists()
            async let a = appModel.service.savedAlbums()
            async let r = appModel.service.recentlyPlayed()
            (playlists, albums, recent) = try await (p, a, r)
            loadState = .loaded
        } catch {
            loadState = .failed((error as? SpotifyError)?.errorDescription ?? error.localizedDescription)
        }
    }
}

// MARK: - Reusable grid card

struct CardView: View {
    let title: String
    let subtitle: String
    let artworkURL: URL?

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            ArtworkView(url: artworkURL)
                .aspectRatio(1, contentMode: .fit)
            Text(title).font(.headline).lineLimit(1)
            Text(subtitle).font(.subheadline).foregroundStyle(.secondary).lineLimit(1)
        }
        .padding(12)
        .background(.thinMaterial, in: RoundedRectangle(cornerRadius: 16, style: .continuous))
        .hoverEffect()
    }
}

enum LoadState: Equatable {
    case loading
    case loaded
    case failed(String)
}
