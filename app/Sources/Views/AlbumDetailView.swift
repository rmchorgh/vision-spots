import SwiftUI

// MARK: - Album detail: header + track list, with a Play button
//
// Mirrors PlaylistDetailView. Album-track listings omit per-track album art, so the album's
// own artwork is passed to each row as a fallback. Tracks page in incrementally.

struct AlbumDetailView: View {
    let album: Album

    @Environment(AppModel.self) private var appModel
    @Environment(PlayerModel.self) private var player

    @State private var tracks: [Track] = []
    @State private var loadState: LoadState = .loading
    @State private var nextOffset: Int?
    @State private var isLoadingMore = false

    private let pageSize = 50

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 24) {
                header
                switch loadState {
                case .loading:
                    ProgressView().controlSize(.large).frame(maxWidth: .infinity, minHeight: 200)
                case .failed(let message):
                    ContentUnavailableView("Couldn't load tracks", systemImage: "exclamationmark.triangle", description: Text(message))
                case .loaded:
                    trackList
                }
            }
            .padding(28)
        }
        .lookToScroll()
        .navigationTitle(album.name)
        .navigationBarTitleDisplayMode(.inline)
        .task { await load() }
    }

    private var header: some View {
        HStack(alignment: .bottom, spacing: 24) {
            ArtworkView(url: album.artworkURL)
                .frame(width: 200, height: 200)
                .shadow(radius: 12, y: 8)
            VStack(alignment: .leading, spacing: 10) {
                Text("Album")
                    .font(.caption.weight(.semibold)).foregroundStyle(.secondary)
                Text(album.name).font(.extraLargeTitle2.weight(.bold))
                Text(album.artistNames).font(.callout).foregroundStyle(.secondary).lineLimit(2)
                Text("\(album.trackCount) tracks")
                    .font(.subheadline).foregroundStyle(.secondary)
                Button {
                    Task { await player.play(contextURI: album.uri) }
                } label: {
                    Label("Play", systemImage: "play.fill").padding(.horizontal, 8)
                }
                .buttonStyle(.borderedProminent)
                .controlSize(.large)
                .padding(.top, 4)
            }
            Spacer()
        }
    }

    private var trackList: some View {
        VStack(spacing: 0) {
            ForEach(Array(tracks.enumerated()), id: \.element.id) { idx, track in
                TrackRow(track: track, index: idx + 1, fallbackArtwork: album.artworkURL)
                    .onTapGesture { Task { await player.play(contextURI: track.uri) } }
                    .onAppear { if track.id == tracks.last?.id { Task { await loadMore() } } }
                Divider().opacity(track.id == tracks.last?.id ? 0 : 1)
            }
            if isLoadingMore {
                ProgressView().controlSize(.regular).frame(maxWidth: .infinity).padding(.vertical, 12)
            }
        }
        .padding(16)
        .background(.thinMaterial, in: RoundedRectangle(cornerRadius: 16, style: .continuous))
    }

    private func load() async {
        loadState = .loading
        do {
            let page = try await appModel.service.albumTracks(id: album.id, offset: 0, limit: pageSize)
            tracks = page.tracks
            nextOffset = page.nextOffset
            loadState = .loaded
        } catch {
            loadState = .failed((error as? SpotifyError)?.errorDescription ?? error.localizedDescription)
        }
    }

    private func loadMore() async {
        guard !isLoadingMore, let offset = nextOffset else { return }
        isLoadingMore = true
        defer { isLoadingMore = false }
        do {
            let page = try await appModel.service.albumTracks(id: album.id, offset: offset, limit: pageSize)
            tracks.append(contentsOf: page.tracks)
            nextOffset = page.nextOffset
        } catch {
            nextOffset = nil
        }
    }
}
