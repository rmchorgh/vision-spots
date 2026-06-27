import SwiftUI

// MARK: - Playlist detail: header + track list, with a Play button

struct PlaylistDetailView: View {
    let playlist: Playlist

    @Environment(AppModel.self) private var appModel
    @Environment(PlayerModel.self) private var player

    @State private var tracks: [Track] = []
    @State private var loadState: LoadState = .loading

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
        .navigationTitle(playlist.name)
        .navigationBarTitleDisplayMode(.inline)
        .task { await load() }
    }

    private var header: some View {
        HStack(alignment: .bottom, spacing: 24) {
            ArtworkView(url: playlist.artworkURL)
                .frame(width: 200, height: 200)
                .shadow(radius: 12, y: 8)
            VStack(alignment: .leading, spacing: 10) {
                Text("Playlist").font(.caption.weight(.semibold)).foregroundStyle(.secondary)
                Text(playlist.name).font(.extraLargeTitle2.weight(.bold))
                Text(playlist.description).font(.callout).foregroundStyle(.secondary).lineLimit(2)
                Text("\(playlist.ownerName) · \(playlist.trackCount) tracks")
                    .font(.subheadline).foregroundStyle(.secondary)
                Button {
                    Task { await player.play(contextURI: playlist.uri) }
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
                TrackRow(track: track, index: idx + 1)
                    .onTapGesture { Task { await player.play(contextURI: track.uri) } }
                if track.id != tracks.last?.id { Divider() }
            }
        }
        .padding(16)
        .background(.thinMaterial, in: RoundedRectangle(cornerRadius: 16, style: .continuous))
    }

    private func load() async {
        loadState = .loading
        do {
            tracks = try await appModel.service.playlistTracks(id: playlist.id)
            loadState = .loaded
        } catch {
            loadState = .failed((error as? SpotifyError)?.errorDescription ?? error.localizedDescription)
        }
    }
}
