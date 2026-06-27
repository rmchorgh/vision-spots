import SwiftUI

// MARK: - Spotify Connect device picker
//
// Lists available Connect devices and transfers playback to the chosen one. `DeviceList`
// holds the rows so it can be embedded (e.g. pushed from the speaker control's volume
// popover); `DevicePickerView` wraps it in its own NavigationStack for standalone use.

struct DevicePickerView: View {
    var body: some View {
        NavigationStack {
            DeviceList()
                .navigationTitle("Connect to a device")
                .navigationBarTitleDisplayMode(.inline)
        }
    }
}

struct DeviceList: View {
    @Environment(PlayerModel.self) private var player
    @Environment(\.dismiss) private var dismiss

    var body: some View {
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
                        Image(systemName: Self.icon(for: device.type))
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
        .padding(.top, 24)
        .task { await player.loadDevices() }
        .frame(minHeight: 200, maxHeight: 260)
    }

    static func icon(for type: String) -> String {
        switch type.lowercased() {
        case "computer":   return "laptopcomputer"
        case "smartphone": return "iphone"
        case "speaker":    return "hifispeaker.fill"
        case "tv":         return "tv"
        default:            return "hifispeaker.2"
        }
    }
}
