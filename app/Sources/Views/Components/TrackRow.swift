import SwiftUI

// MARK: - A single track row (track list / search results)

struct TrackRow: View {
    let track: Track
    var index: Int? = nil

    var body: some View {
        HStack(spacing: 14) {
            if let index {
                Text("\(index)")
                    .font(.subheadline.monospacedDigit())
                    .foregroundStyle(.secondary)
                    .frame(width: 28, alignment: .trailing)
            }
            ArtworkView(url: track.artworkURL, cornerRadius: 6)
                .frame(width: 48, height: 48)
            VStack(alignment: .leading, spacing: 2) {
                Text(track.name).font(.body).lineLimit(1)
                Text(track.artistNames).font(.subheadline).foregroundStyle(.secondary).lineLimit(1)
            }
            Spacer()
            Text(track.durationFormatted)
                .font(.subheadline.monospacedDigit())
                .foregroundStyle(.secondary)
        }
        .padding(.vertical, 4)
        .contentShape(Rectangle())
    }
}
