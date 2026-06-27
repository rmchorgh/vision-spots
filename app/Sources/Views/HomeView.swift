import SwiftUI

// MARK: - Home: hero slider + recently played
//
// A paged hero carousel (the current Daylist, then Liked Songs) sits above a grid of the
// last-played albums and playlists (excluding the daylist and Liked Songs).

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
        .navigationTitle("Home")
        .task { await load() }
    }

    private var content: some View {
        VStack(alignment: .leading, spacing: 36) {
            HeroCarousel(daylist: daylist, liked: liked)

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

// MARK: - Hero carousel

private struct HeroCarousel: View {
    let daylist: Playlist?
    let liked: Playlist?

    @State private var page = 0

    private var pages: [Playlist] { [daylist, liked].compactMap { $0 } }

    var body: some View {
        TabView(selection: $page) {
            ForEach(Array(pages.enumerated()), id: \.element.id) { index, playlist in
                NavigationLink(value: playlist) {
                    HeroCard(playlist: playlist)
                }
                .buttonStyle(.plain)
                .tag(index)
                .padding(.bottom, 44)   // leave room for the page dots
            }
        }
        .tabViewStyle(.page(indexDisplayMode: .always))
        .frame(height: 380)
    }
}

private struct HeroCard: View {
    let playlist: Playlist

    var body: some View {
        ZStack(alignment: .bottomLeading) {
            artwork
                .frame(maxWidth: .infinity, maxHeight: .infinity)
                .clipped()

            // Scrim: a top-down darkening plus a denser pad behind the text for legibility
            // regardless of the underlying artwork's brightness.
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
