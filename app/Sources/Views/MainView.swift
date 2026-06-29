import SwiftUI

// MARK: - Main app shell: adaptable sidebar + a Now Playing ornament
//
// Uses the visionOS `.sidebarAdaptable` TabView style — the floating glass sidebar (Home,
// Playlists, Library, Search, Settings) seen in Apple Music for Vision Pro.

struct MainView: View {
    @Environment(PlayerModel.self) private var player

    enum Screen: Hashable { case home, playlists, library, search, settings }

    @State private var selection: Screen = .home

    var body: some View {
        TabView(selection: $selection) {
            Tab("Home", systemImage: "house", value: .home) {
                stack { HomeView() }
            }
            Tab("Playlists", systemImage: "music.note.list", value: .playlists) {
                stack { PlaylistsView() }
            }
            Tab("Library", systemImage: "square.stack", value: .library) {
                stack { LibraryView() }
            }
            Tab("Search", systemImage: "magnifyingglass", value: .search) {
                stack { SearchView() }
            }
            Tab("Settings", systemImage: "gearshape", value: .settings) {
                stack { SettingsView() }
            }
        }
        .tabViewStyle(.sidebarAdaptable)
        .ornament(attachmentAnchor: .scene(.bottom)) {
            NowPlayingBar()
                .padding(.bottom, 12)
        }
        .task {
            await player.loadDevices()
            await player.startLivePlaybackUpdates()   // runs until this view disappears
        }
    }

    /// Each tab gets its own navigation stack so playlists and albums can push a detail view.
    private func stack<Content: View>(@ViewBuilder _ content: () -> Content) -> some View {
        NavigationStack {
            content()
                .navigationDestination(for: Playlist.self) { PlaylistDetailView(playlist: $0) }
                .navigationDestination(for: Album.self) { AlbumDetailView(album: $0) }
        }
    }
}
