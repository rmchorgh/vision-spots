import SwiftUI

// MARK: - Async artwork with placeholder
//
// Loads album/playlist art, showing a tinted placeholder while loading or on failure.

struct ArtworkView: View {
    let url: URL?
    var cornerRadius: CGFloat = 12

    var body: some View {
        AsyncImage(url: url) { phase in
            switch phase {
            case .success(let image):
                image.resizable().scaledToFill()
            case .failure:
                placeholder(systemImage: "music.note")
            case .empty:
                placeholder(systemImage: nil)
            @unknown default:
                placeholder(systemImage: "music.note")
            }
        }
        .clipShape(RoundedRectangle(cornerRadius: cornerRadius, style: .continuous))
    }

    private func placeholder(systemImage: String?) -> some View {
        ZStack {
            RoundedRectangle(cornerRadius: cornerRadius, style: .continuous)
                .fill(.quaternary)
            if let systemImage {
                Image(systemName: systemImage)
                    .font(.largeTitle)
                    .foregroundStyle(.secondary)
            } else {
                ProgressView()
            }
        }
    }
}
