import SwiftUI

// MARK: - Now Playing ornament (bottom of the main window)
//
// Three zones, matching Apple Music for Vision Pro: transport controls on the left, a
// "sunken" recessed media-status pill in the middle, and the speaker controller on the right.

struct NowPlayingBar: View {
    @Environment(PlayerModel.self) private var player

    var body: some View {
        HStack(spacing: 20) {
            transportControls
            mediaStatus
            SpeakerControl()
        }
        .padding(.horizontal, 22)
        .padding(.vertical, 12)
        .frame(width: 760)
        .glassBackgroundEffect()
    }

    private var transportControls: some View {
        HStack(spacing: 18) {
            controlButton("backward.fill") { Task { await player.previous() } }
            controlButton(player.state.isPlaying ? "pause.fill" : "play.fill", large: true) {
                Task { await player.togglePlayPause() }
            }
            controlButton("forward.fill") { Task { await player.next() } }
        }
    }

    // The recessed center pill — a darker, inset fill reads as "sunken" against the glass.
    private var mediaStatus: some View {
        HStack(spacing: 12) {
            ArtworkView(url: player.state.track?.artworkURL, cornerRadius: 8)
                .frame(width: 44, height: 44)
            VStack(alignment: .leading, spacing: 2) {
                Text(player.state.track?.name ?? "Nothing playing")
                    .font(.subheadline.weight(.semibold)).lineLimit(1)
                Text(player.state.track?.artistNames ?? "Pick a song to start")
                    .font(.caption).foregroundStyle(.secondary).lineLimit(1)
            }
            Spacer(minLength: 0)
        }
        .padding(.horizontal, 12)
        .padding(.vertical, 8)
        .frame(maxWidth: .infinity)
        .background(
            RoundedRectangle(cornerRadius: 14, style: .continuous)
                .fill(.black.opacity(0.22))
        )
        .overlay(
            RoundedRectangle(cornerRadius: 14, style: .continuous)
                .strokeBorder(.white.opacity(0.06), lineWidth: 1)
        )
    }

    private func controlButton(_ systemName: String, large: Bool = false, action: @escaping () -> Void) -> some View {
        Button(action: action) {
            Image(systemName: systemName)
                .font(large ? .title2 : .title3)
                .frame(width: large ? 48 : 38, height: large ? 48 : 38)
        }
        .buttonStyle(.plain)
        .disabled(!player.hasTrack)
    }
}
