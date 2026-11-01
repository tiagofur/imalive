import SwiftUI

/// App root: the session gate (issue #76 slice 2) decides what renders —
/// the onboarding flow without a session, home with one. The OTP verify
/// (slice 3) saves the session and the gate swaps the flow for home
/// reactively — no navigation call involved; a session wiped by the
/// refresher (a rejected refresh) walks the user back the same way.
struct ContentView: View {
    @Environment(HomeModel.self) private var home
    @Environment(OnboardingModel.self) private var onboarding

    var body: some View {
        switch home.gate {
        case .loading:
            // The store answers at init; a blank frame at most (no spinner).
            Color.clear
        case .needsOnboarding:
            OnboardingScreen(model: onboarding)
        case .ready:
            HomeView(model: home)
        }
    }
}
