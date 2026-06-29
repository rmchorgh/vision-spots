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
            MediaStatus()
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

// MARK: - Center media pill (recessed, "sunken" against the glass)
//
// Title + artist on the left, elapsed / total time on the right, and a hairline scrubber pinned
// to the bottom edge (Apple Music style). On touch the bar swells and the time fades in, then it
// seeks on release. Optimistic local progress (PlayerModel) keeps it moving between polls.
//
// Performance notes — this view updates ~1×/s (the live tick) and ~60×/s while dragging, all
// inside a `.glassBackgroundEffect()` ornament, so it's deliberately cheap:
//   • Layout height is CONSTANT: the time fades via opacity (it never re-flows the row) and the
//     bar's swell happens inside a fixed-height hit area, so the pill never grows and the glass
//     never re-rasterizes.
//   • Width is captured once via `onGeometryChange` rather than a per-frame `GeometryReader`.
//   • The swell is a scoped `withAnimation`; the fill follows the finger with no animation while
//     dragging, and glides with a 1 s linear animation between polls when idle.

private struct MediaStatus: View {
    @Environment(PlayerModel.self) private var player

    /// 0…1 drag position; non-nil only while the user is actively scrubbing.
    @State private var dragFraction: Double?
    /// Cached track width, so the drag math never needs a `GeometryReader`.
    @State private var trackWidth: CGFloat = 0
    /// Drives the swell + time fade. Animated explicitly on scrub start/end.
    @State private var swollen = false

    private var durationMs: Double { Double(player.state.track?.durationMs ?? 0) }

    var body: some View {
        let duration = max(durationMs, 1)
        let liveFraction = min(Double(player.state.progressMs) / duration, 1)
        let scrubbing = dragFraction != nil
        let fraction = dragFraction ?? liveFraction
        let positionMs = fraction * duration

        VStack(spacing: 0) {
            HStack(spacing: 12) {
                ArtworkView(url: player.state.track?.artworkURL, cornerRadius: 8)
                    .frame(width: 44, height: 44)
                VStack(alignment: .leading, spacing: 2) {
                    Text(player.state.track?.name ?? "Nothing playing")
                        .font(.subheadline.weight(.semibold)).lineLimit(1)
                    Text(player.state.track?.artistNames ?? "Pick a song to start")
                        .font(.caption).foregroundStyle(.secondary).lineLimit(1)
                }
                Spacer(minLength: 8)
                /*
                    Elapsed / total — fades in on scrub but always holds its place, so the row
                    never reflows.
                */
                Text("\(timeLabel(positionMs)) / \(timeLabel(duration))")
                    .font(.caption2.monospacedDigit())
                    .foregroundStyle(.secondary)
                    .opacity(player.hasTrack ? 1 : 0)
            }
            
            if player.hasTrack {
                seekBar(fraction: fraction, scrubbing: scrubbing, duration: duration)
            }
        }
        .padding(.horizontal)
        .padding(.top, 8)
        .frame(maxWidth: .infinity)
        .background(
            RoundedRectangle(cornerRadius: 14, style: .continuous)
                .fill(.black.opacity(0.22))
        )
        .disabled(!player.hasTrack)
    }

    private func seekBar(fraction: Double, scrubbing: Bool, duration: Double) -> some View {
        ZStack {
            ZStack(alignment: .leading) {
                Capsule().fill(.white.opacity(0.18))
                Capsule().fill(.white.opacity(0.9))
                    .frame(width: max(0, trackWidth * fraction))
            }
            .frame(height: swollen ? 6 : 2)
            // Glide between polls when idle; track the finger instantly while scrubbing.
            .animation(scrubbing ? nil : .linear(duration: 1), value: fraction)
        }
        .frame(height: 12, alignment: .bottom)              // fixed hit area; bar centers within it
        .contentShape(Rectangle())                          // whole strip is draggable / tappable
        .onGeometryChange(for: CGFloat.self) { $0.size.width } action: { trackWidth = $0 }
        .gesture(
            DragGesture(minimumDistance: 0)
                .onChanged { value in
                    if !swollen {
                        if !player.isSeeking { player.isSeeking = true }
                        withAnimation(.easeOut(duration: 0.18)) { swollen = true }
                    }
                    dragFraction = clampedFraction(value.location.x)
                }
                .onEnded { value in
                    let target = Int(clampedFraction(value.location.x) * duration)
                    dragFraction = nil
                    player.isSeeking = false
                    withAnimation(.easeOut(duration: 0.18)) { swollen = false }
                    Task { await player.seek(toPositionMs: target) }
                }
        )
    }

    private func clampedFraction(_ x: CGFloat) -> Double {
        guard trackWidth > 0 else { return 0 }
        return min(max(0, Double(x / trackWidth)), 1)
    }

    private func timeLabel(_ ms: Double) -> String {
        let totalSeconds = Int(max(0, ms)) / 1000
        return String(format: "%d:%02d", totalSeconds / 60, totalSeconds % 60)
    }
}
