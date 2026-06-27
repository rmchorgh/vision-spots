import SwiftUI

// MARK: - Settings: account info + sign out

struct SettingsView: View {
    @Environment(AppModel.self) private var appModel

    var body: some View {
        Form {
            Section("Account") {
                HStack(spacing: 16) {
                    ArtworkView(url: appModel.currentUser?.imageURL, cornerRadius: 30)
                        .frame(width: 60, height: 60)
                    VStack(alignment: .leading, spacing: 2) {
                        Text(appModel.currentUser?.displayName ?? "—")
                            .font(.headline)
                        Text((appModel.currentUser?.isPremium ?? false) ? "Spotify Premium" : "Spotify Free")
                            .font(.subheadline).foregroundStyle(.secondary)
                    }
                }
                .padding(.vertical, 4)
            }

            Section {
                Button(role: .destructive) {
                    Task { await appModel.signOut() }
                } label: {
                    Label("Sign Out", systemImage: "rectangle.portrait.and.arrow.right")
                }
            }
        }
        .navigationTitle("Settings")
        .navigationBarTitleDisplayMode(.inline)
    }
}
