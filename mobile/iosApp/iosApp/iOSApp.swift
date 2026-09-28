import SwiftUI
import Shared

@main
struct CarteroApp: App {
    init() {
        CarteroIos.shared.start()
    }

    var body: some Scene {
        WindowGroup {
            ContentView()
        }
    }
}
