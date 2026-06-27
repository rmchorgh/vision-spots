import Foundation
import Observation

// MARK: - Playback state (Spotify Connect)
//
// Shared by the Now Playing bar and the device picker. Wraps the SpotifyService player calls
// and keeps a lightweight polled snapshot of what's playing. Maps to backend /api/player/*.

@MainActor
@Observable
final class PlayerModel {

    private let service: any SpotifyService

    var state: PlaybackState = .idle
    var devices: [Device] = []
    var errorMessage: String?

    init(service: any SpotifyService) {
        self.service = service
    }

    var hasTrack: Bool { state.track != nil }

    func refresh() async {
        do { state = try await service.playbackState() }
        catch { handle(error) }
    }

    func loadDevices() async {
        do { devices = try await service.devices() }
        catch { handle(error) }
    }

    func togglePlayPause() async {
        do {
            if state.isPlaying { try await service.pause() } else { try await service.play(contextURI: nil, deviceID: nil) }
            state.isPlaying.toggle()
        } catch { handle(error) }
    }

    func next() async {
        do { try await service.next(); await refresh() } catch { handle(error) }
    }

    func previous() async {
        do { try await service.previous(); await refresh() } catch { handle(error) }
    }

    func play(contextURI: String) async {
        do { try await service.play(contextURI: contextURI, deviceID: nil); await refresh() }
        catch { handle(error) }
    }

    func transfer(to device: Device) async {
        do { try await service.transferPlayback(toDeviceID: device.id); await loadDevices(); await refresh() }
        catch { handle(error) }
    }

    /// Is there an active Connect device we can show a volume bar for?
    var hasDevice: Bool { state.device != nil }

    /// Current device volume (0–100), defaulting to 50 when unknown.
    var volume: Int { state.device?.volumePercent ?? 50 }

    /// Sets the active device volume. Updates local state optimistically so the slider stays
    /// responsive, then tells the backend.
    func setVolume(_ percent: Int) async {
        let clamped = min(100, max(0, percent))
        state.device?.volumePercent = clamped
        if let i = devices.firstIndex(where: { $0.isActive }) {
            devices[i].volumePercent = clamped
        }
        do { try await service.setVolume(percent: clamped) } catch { handle(error) }
    }

    private func handle(_ error: Error) {
        errorMessage = (error as? SpotifyError)?.errorDescription ?? error.localizedDescription
    }
}
