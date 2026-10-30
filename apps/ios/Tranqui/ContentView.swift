import SwiftUI

/// F0 placeholder root: proves the shell builds and launches. The second PR of
/// the chain (issue #4) replaces this with the real home (HomeView).
struct ContentView: View {
    var body: some View {
        NavigationStack {
            Text("Tranqui")
                .navigationTitle("Tranqui")
        }
    }
}
