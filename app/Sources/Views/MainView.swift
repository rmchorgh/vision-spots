import SwiftUI

// MARK: - Main app shell: sidebar + detail, with a Now Playing ornament

struct MainView: View {
    @Environment(AppModel.self) private var appModel
    @Environment(PlayerModel.self) private var player

    enum Tab: Hashable { case library, search }

    @State private var selection: Tab? = .library
    @State private var path = NavigationPath()

    var body: some View {
        NavigationSplitView {
            sidebar
        } detail: {
            NavigationStack(path: $path) {
                detail
                    .navigationDestination(for: Playlist.self) { PlaylistDetailView(playlist: $0) }
            }
        }
        .ornament(attachmentAnchor: .scene(.bottom)) {
            NowPlayingBar()
                .padding(.bottom, 12)
        }
        .task {
            await player.refresh()
            await player.loadDevices()
        }
    }

    private var sidebar: some View {
        List(selection: $selection) {
            Label("Library", systemImage: "square.stack").tag(Tab.library)
            Label("Search", systemImage: "magnifyingglass").tag(Tab.search)

            Spacer(minLength: 24)

            Section {
                HStack(spacing: 12) {
                    ArtworkView(url: appModel.currentUser?.imageURL, cornerRadius: 18)
                        .frame(width: 36, height: 36)
                    VStack(alignment: .leading, spacing: 0) {
                        Text(appModel.currentUser?.displayName ?? "—").lineLimit(1)
                        Text((appModel.currentUser?.isPremium ?? false) ? "Premium" : "Free")
                            .font(.caption).foregroundStyle(.secondary)
                    }
                }
                Button(role: .destructive) {
                    Task { await appModel.signOut() }
                } label: {
                    Label("Sign Out", systemImage: "rectangle.portrait.and.arrow.right")
                }
            }
        }
        .navigationTitle("vision-spots")
    }

    @ViewBuilder
    private var detail: some View {
        switch selection ?? .library {
        case .library: LibraryView()
        case .search:  SearchView()
        }
    }
}
