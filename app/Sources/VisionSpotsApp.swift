import SwiftUI

// MARK: - App entry point
//
// One WindowGroup hosting RootView. AppModel + PlayerModel are created here and shared via
// the SwiftUI environment. Which SpotifyService backs them is decided by AppConfig.makeService().

@main
struct VisionSpotsApp: App {

    @State private var appModel: AppModel
    @State private var player: PlayerModel

    init() {
        let model = AppModel()
        _appModel = State(initialValue: model)
        _player = State(initialValue: PlayerModel(service: model.service))
    }

    var body: some Scene {
        WindowGroup {
            RootView()
                .environment(appModel)
                .environment(player)
                .frame(minWidth: 900, minHeight: 900)
        }
        .defaultSize(width: 1100, height: 760)
    }
}
