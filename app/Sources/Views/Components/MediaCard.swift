import SwiftUI

// MARK: - Reusable album / playlist card
//
// One component for both kinds of library item (see `MediaItem`). Playlists push a detail
// view; albums start playback on tap. Used by the Home grid, Playlists grid, and search.

struct MediaCard: View {
    let item: MediaItem

    @Environment(PlayerModel.self) private var player

    var body: some View {
        switch item {
        case .playlist(let playlist):
            NavigationLink(value: playlist) {
                cardBody(artworkURL: playlist.artworkURL,
                         likedSongs: playlist.isLikedSongs,
                         title: playlist.name,
                         subtitle: subtitle(for: playlist))
            }
            .buttonStyle(.plain)

        case .album(let album):
            Button {
                Task { await player.play(contextURI: album.uri) }
            } label: {
                cardBody(artworkURL: album.artworkURL,
                         likedSongs: false,
                         title: album.name,
                         subtitle: album.artistNames)
            }
            .buttonStyle(.plain)
        }
    }

    private func subtitle(for playlist: Playlist) -> String {
        playlist.description.isEmpty ? "\(playlist.trackCount) songs" : playlist.description
    }

    private func cardBody(artworkURL: URL?, likedSongs: Bool, title: String, subtitle: String) -> some View {
        VStack(alignment: .leading, spacing: 10) {
            Group {
                if likedSongs {
                    LikedSongsArtwork()
                } else {
                    ArtworkView(url: artworkURL)
                }
            }
            .aspectRatio(1, contentMode: .fit)
            .shadow(radius: 8, y: 4)

            VStack(alignment: .leading, spacing: 2) {
                Text(title).font(.headline).lineLimit(1)
                Text(subtitle).font(.subheadline).foregroundStyle(.secondary).lineLimit(1)
            }
        }
        .padding(12)
        .background(.thinMaterial, in: RoundedRectangle(cornerRadius: 18, style: .continuous))
        .hoverEffect()
    }
}

// MARK: - Liked Songs gradient artwork
//
// Liked Songs has no real cover art; Spotify renders a purple→blue gradient with a heart.
// Reused by the card and the Home hero.

struct LikedSongsArtwork: View {
    var cornerRadius: CGFloat = 12
    var heartFont: Font = .largeTitle

    var body: some View {
        ZStack {
            LinearGradient(
                colors: [Color(red: 0.42, green: 0.20, blue: 0.86),
                         Color(red: 0.29, green: 0.51, blue: 0.96)],
                startPoint: .topLeading, endPoint: .bottomTrailing)
            Image(systemName: "heart.fill")
                .font(heartFont)
                .foregroundStyle(.white)
        }
        .clipShape(RoundedRectangle(cornerRadius: cornerRadius, style: .continuous))
    }
}
