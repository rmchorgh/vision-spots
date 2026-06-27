import SwiftUI

// MARK: - Home: daylist + liked songs side-by-side + recently played
//
// The two featured playlists (daylist and liked songs) sit side-by-side taking 50% of
// the available width each. They use the same 380pt min height as the previous hero.
// Below is the grid of recently played items (excluding daylist and Liked Songs).

struct HomeView: View {
    @Environment(AppModel.self) private var appModel

    @State private var daylist: Playlist?
    @State private var liked: Playlist?
    @State private var recent: [MediaItem] = []
    @State private var loadState: LoadState = .loading

    private let columns = [GridItem(.adaptive(minimum: 180, maximum: 220), spacing: 24)]

    var body: some View {
        ScrollView {
            switch loadState {
            case .loading:
                ProgressView().controlSize(.large).frame(maxWidth: .infinity, minHeight: 400)
            case .failed(let message):
                ContentUnavailableView("Couldn't load Home", systemImage: "exclamationmark.triangle",
                                       description: Text(message))
                    .frame(minHeight: 400)
            case .loaded:
                content
            }
        }
        .padding(.bottom, 96)
        .navigationTitle("Home")
        .task { await load() }
    }

    private var content: some View {
        VStack(alignment: .leading, spacing: 36) {
            HStack(spacing: 24) {
                if let daylist {
                    NavigationLink(value: daylist) {
                        HeroCard(playlist: daylist)
                    }
                    .buttonStyle(.plain)
                    .frame(maxWidth: .infinity, minHeight: 380)
                }
                if let liked {
                    NavigationLink(value: liked) {
                        HeroCard(playlist: liked)
                    }
                    .buttonStyle(.plain)
                    .frame(maxWidth: .infinity, minHeight: 380)
                }
            }

            if !recent.isEmpty {
                VStack(alignment: .leading, spacing: 18) {
                    Text("Recently Played").font(.title.weight(.bold))
                    LazyVGrid(columns: columns, spacing: 24) {
                        ForEach(recent) { MediaCard(item: $0) }
                    }
                }
            }
        }
        .padding(28)
    }

    private func load() async {
        loadState = .loading
        do {
            async let d = appModel.service.daylist()
            async let l = appModel.service.likedSongs()
            async let r = appModel.service.recentlyPlayed()
            (daylist, liked, recent) = try await (d, l, r)
            loadState = .loaded
        } catch {
            loadState = .failed((error as? SpotifyError)?.errorDescription ?? error.localizedDescription)
        }
    }
}

private struct HeroCard: View {
    let playlist: Playlist

    var body: some View {
        ZStack(alignment: .bottomLeading) {
            artwork
                .frame(maxWidth: .infinity, maxHeight: .infinity)
                .clipped()

            LinearGradient(colors: [.black.opacity(0.0), .black.opacity(0.25), .black.opacity(0.85)],
                           startPoint: .top, endPoint: .bottom)

            VStack(alignment: .leading, spacing: 8) {
                Text(playlist.isDaylist ? "YOUR DAYLIST" : "PLAYLIST")
                    .font(.caption.weight(.bold))
                    .foregroundStyle(.white.opacity(0.9))
                Text(playlist.name)
                    .font(.extraLargeTitle2.weight(.heavy))
                    .foregroundStyle(.white)
                    .lineLimit(2)
                if !playlist.description.isEmpty {
                    Text(playlist.description)
                        .font(.title3)
                        .foregroundStyle(.white.opacity(0.92))
                        .lineLimit(2)
                }
            }
            .shadow(color: .black.opacity(0.6), radius: 6, y: 2)
            .padding(28)
            .frame(maxWidth: .infinity, alignment: .leading)
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity)
        .clipShape(RoundedRectangle(cornerRadius: 28, style: .continuous))
        .hoverEffect()
    }

    @ViewBuilder
    private var artwork: some View {
        if playlist.isLikedSongs {
            LikedSongsArtwork(cornerRadius: 0, heartFont: .system(size: 80))
        } else {
            ArtworkView(url: playlist.artworkURL, cornerRadius: 0)
        }
    }
}
