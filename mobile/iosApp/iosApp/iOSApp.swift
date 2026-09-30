import SwiftUI
import Shared

@main
struct CarteroApp: App {
    init() {
        CarteroIos.shared.start(iroh: IrohNetwork())
    }

    var body: some Scene {
        WindowGroup {
            ContentView()
        }
    }
}
