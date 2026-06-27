import SwiftUI

// MARK: - Speaker controller (right side of the Now Playing bar)
//
// Tap behaviour mirrors Apple Music for Vision Pro:
//   • a device is connected  → popover with a volume bar + "Choose a different speaker"
//   • nothing connected      → open the device picker straight away

struct SpeakerControl: View {
    @Environment(PlayerModel.self) private var player

    @State private var showVolume = false
    @State private var showPicker = false

    var body: some View {
        Button {
            if player.hasDevice { showVolume = true } else { showPicker = true }
        } label: {
            Image(systemName: "hifispeaker.2.fill")
                .font(.title3)
                .frame(width: 44, height: 44)
        }
        .buttonStyle(.plain)
        .popover(isPresented: $showVolume) {
            VolumePopover().frame(minWidth: 320)
        }
        .popover(isPresented: $showPicker) {
            DevicePickerView().frame(minWidth: 320, minHeight: 260)
        }
    }
}

private struct VolumePopover: View {
    @Environment(PlayerModel.self) private var player
    @State private var volume: Double = 50

    var body: some View {
        NavigationStack {
            VStack(alignment: .leading, spacing: 16) {
                if let device = player.state.device {
                    Label(device.name, systemImage: DeviceList.icon(for: device.type))
                        .font(.headline)
                }

                HStack(spacing: 14) {
                    Image(systemName: "speaker.fill").foregroundStyle(.secondary)
                    Slider(value: $volume, in: 0...100) { editing in
                        if !editing { Task { await player.setVolume(Int(volume)) } }
                    }
                    Image(systemName: "speaker.wave.3.fill").foregroundStyle(.secondary)
                }

                NavigationLink {
                    DeviceList()
                        .navigationTitle("Connect to a device")
                        .navigationBarTitleDisplayMode(.inline)
                } label: {
                    Label("Choose a Different Speaker", systemImage: "hifispeaker.2")
                        .frame(maxWidth: .infinity)
                }
                .buttonStyle(.bordered)
            }
            .padding(16)
        }
        .onAppear { volume = Double(player.volume) }
    }
}
