import SwiftUI
import UIKit
import Shared

struct ComposeView: UIViewControllerRepresentable {
    func makeUIViewController(context: Context) -> UIViewController {
        CarteroIos.shared.mainViewController()
    }

    func updateUIViewController(_ uiViewController: UIViewController, context: Context) {}
}

struct ContentView: View {
    @State private var barsHidden = false

    var body: some View {
        ComposeView()
            .ignoresSafeArea()
            .preferredColorScheme(.dark)
            .statusBarHidden(barsHidden)
            .persistentSystemOverlays(barsHidden ? .hidden : .automatic)
            .onAppear {
                CarteroIos.shared.observeSystemBars { hidden in
                    withAnimation(.easeInOut(duration: 0.2)) { barsHidden = hidden.boolValue }
                }
            }
    }
}
