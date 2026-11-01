import SwiftUI

/// App root: the check-in home (the only screen until the onboarding gate
/// lands; issue #76 slice 3 switches on the session gate here).
struct ContentView: View {
    @Environment(HomeModel.self) private var home

    var body: some View {
        HomeView(model: home)
    }
}
