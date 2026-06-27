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
        ZStack(alignment: .bottomLeading) {
            Group {
                if likedSongs {
                    LikedSongsArtwork(cornerRadius: 0)
                } else {
                    ArtworkView(url: artworkURL, cornerRadius: 0)
                }
            }
            .frame(maxWidth: .infinity, maxHeight: .infinity)
            .clipped()

            LinearGradient(colors: [.black.opacity(0.0), .black.opacity(0.25), .black.opacity(0.85)],
                           startPoint: .top, endPoint: .bottom)

            VStack(alignment: .leading, spacing: 4) {
                Text(title)
                    .font(.headline.weight(.bold))
                    .foregroundStyle(.white)
                    .lineLimit(2)
                if !subtitle.isEmpty {
                    Text(subtitle)
                        .font(.subheadline)
                        .foregroundStyle(.white.opacity(0.85))
                        .lineLimit(1)
                }
            }
            .shadow(color: .black.opacity(0.6), radius: 4, y: 1)
            .padding(14)
            .frame(maxWidth: .infinity, alignment: .leading)
        }
        .aspectRatio(1, contentMode: .fit)
        .clipShape(RoundedRectangle(cornerRadius: 18, style: .continuous))
        .shadow(radius: 8, y: 4)
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
