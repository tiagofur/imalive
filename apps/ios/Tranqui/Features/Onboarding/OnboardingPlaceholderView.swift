import SwiftUI

/// Entry point of the onboarding while the gate has no session (issue #76
/// slice 2). Slice 3 replaces this placeholder with the real welcome step;
/// the strings are already the brief's copy (ux-brief, paso 1), so only the
/// body of the flow changes under the same gate.
struct OnboardingPlaceholderView: View {
    var body: some View {
        VStack(spacing: 24) {
            Text(String(localized: "onboarding.welcome_title"))
                .font(.headline)
            Button(String(localized: "onboarding.welcome_cta")) {
                // Slice 3 mounts the real flow under this gate.
            }
            .buttonStyle(.borderedProminent)
        }
        .padding()
    }
}
