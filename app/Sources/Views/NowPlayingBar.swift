import SwiftUI

// MARK: - Now Playing ornament (bottom of the main window)
//
// Shows the current track and transport controls, plus a Spotify Connect device picker.
// Controls map to PlayerModel → SpotifyService → backend /api/player/*.

struct NowPlayingBar: View {
    @Environment(PlayerModel.self) private var player
    @State private var showDevicePicker = false

    var body: some View {
        HStack(spacing: 18) {
            nowPlayingInfo

            Spacer(minLength: 24)

            HStack(spacing: 22) {
                controlButton("backward.fill") { Task { await player.previous() } }
                controlButton(player.state.isPlaying ? "pause.fill" : "play.fill", large: true) {
                    Task { await player.togglePlayPause() }
                }
                controlButton("forward.fill") { Task { await player.next() } }
            }

            Spacer(minLength: 24)

            Button {
                showDevicePicker = true
            } label: {
                Label(player.state.device?.name ?? "Devices", systemImage: "hifispeaker.2.fill")
                    .lineLimit(1)
            }
            .popover(isPresented: $showDevicePicker) {
                DevicePickerView().frame(minWidth: 320, minHeight: 260)
            }
        }
        .padding(.horizontal, 22)
        .padding(.vertical, 14)
        .frame(width: 760)
        .glassBackgroundEffect()
    }

    private var nowPlayingInfo: some View {
        HStack(spacing: 12) {
            ArtworkView(url: player.state.track?.artworkURL, cornerRadius: 8)
                .frame(width: 52, height: 52)
            VStack(alignment: .leading, spacing: 2) {
                Text(player.state.track?.name ?? "Nothing playing")
                    .font(.headline).lineLimit(1)
                Text(player.state.track?.artistNames ?? "Pick a song to start")
                    .font(.subheadline).foregroundStyle(.secondary).lineLimit(1)
            }
        }
        .frame(width: 240, alignment: .leading)
    }

    private func controlButton(_ systemName: String, large: Bool = false, action: @escaping () -> Void) -> some View {
        Button(action: action) {
            Image(systemName: systemName)
                .font(large ? .title : .title3)
                .frame(width: large ? 52 : 40, height: large ? 52 : 40)
        }
        .buttonStyle(.plain)
        .disabled(!player.hasTrack)
    }
}
