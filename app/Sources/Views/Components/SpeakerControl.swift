import SwiftUI

// MARK: - Speaker controller (right side of the Now Playing bar)
//
// Tap behaviour mirrors Apple Music for Vision Pro:
//   • a device is connected  → popover with a volume bar + "Choose a different speaker"
//   • nothing connected      → open the device picker straight away
//
// Both states live in a single popover that swaps its content; presenting two separate
// popovers from the same view and handing off between them renders unreliably.

struct SpeakerControl: View {
    @Environment(PlayerModel.self) private var player
    
    @State private var showPopover = false
    
    var body: some View {
        Button {
            showPopover = true
        } label: {
            Image(systemName: "hifispeaker.2.fill")
                .font(.title3)
                .frame(width: 44, height: 44)
        }
        .buttonStyle(.plain)
        .popover(isPresented: $showPopover) {
            SpeakerPopover(startInDeviceList: !player.hasDevice)
                .frame(minWidth: 320)
        }
    }
}

private struct SpeakerPopover: View {
    @Environment(PlayerModel.self) private var player
    @State private var volume: Double = 50
    @State private var showingDevices: Bool
    
    init(startInDeviceList: Bool) {
        _showingDevices = State(initialValue: startInDeviceList)
    }
    
    var body: some View {
        if showingDevices {
            // A List has no intrinsic height, so it can't drive the popover to resize
            // on its own — pin an explicit height so the box grows to fit the list.
            DeviceList()
                .navigationTitle("Connect to a device")
        } else {
            volumeControls
        }
    }
    
    private var volumeControls: some View {
        VStack(alignment: .leading, spacing: 16) {
            if let device = player.state.device {
                HStack(spacing: 8) {
                    Label(device.name, systemImage: DeviceList.icon(for: device.type))
                        .font(.headline)
                    Text("\(Int(volume))%")
                        .font(.headline)
                        .foregroundStyle(.secondary)   // dimmed, tracks the slider live
                }
            }
            
            HStack(spacing: 14) {
                Image(systemName: "speaker.fill").foregroundStyle(.secondary)
                Slider(value: $volume, in: 0...100) { editing in
                    if !editing { Task { await player.setVolume(Int(volume)) } }
                }
                Image(systemName: "speaker.wave.3.fill").foregroundStyle(.secondary)
            }
            
            Button {
                showingDevices = true
            } label: {
                Label("Choose a Different Speaker", systemImage: "hifispeaker.2")
                    .frame(maxWidth: .infinity)
            }
            .buttonStyle(.bordered)
        }
        .padding(.all)
        .onAppear { volume = Double(player.volume) }
    }
}
