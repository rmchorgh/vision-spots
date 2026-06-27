import SwiftUI

// MARK: - Spotify Connect device picker
//
// Lists available Connect devices and transfers playback to the chosen one.

struct DevicePickerView: View {
    @Environment(PlayerModel.self) private var player
    @Environment(\.dismiss) private var dismiss

    var body: some View {
        NavigationStack {
            List {
                if player.devices.isEmpty {
                    ContentUnavailableView("No devices", systemImage: "hifispeaker",
                                           description: Text("Open Spotify on a phone, speaker, or computer."))
                }
                ForEach(player.devices) { device in
                    Button {
                        Task { await player.transfer(to: device); dismiss() }
                    } label: {
                        HStack {
                            Image(systemName: icon(for: device.type))
                                .frame(width: 28)
                            VStack(alignment: .leading) {
                                Text(device.name)
                                if let vol = device.volumePercent {
                                    Text("Volume \(vol)%").font(.caption).foregroundStyle(.secondary)
                                }
                            }
                            Spacer()
                            if device.isActive {
                                Image(systemName: "checkmark.circle.fill").foregroundStyle(.tint)
                            }
                        }
                    }
                    .buttonStyle(.plain)
                }
            }
            .navigationTitle("Connect to a device")
            .navigationBarTitleDisplayMode(.inline)
            .task { await player.loadDevices() }
        }
    }

    private func icon(for type: String) -> String {
        switch type.lowercased() {
        case "computer":   return "laptopcomputer"
        case "smartphone": return "iphone"
        case "speaker":    return "hifispeaker.fill"
        case "tv":         return "tv"
        default:            return "hifispeaker.2"
        }
    }
}
