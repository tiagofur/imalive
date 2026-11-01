import SwiftUI

/// Root of the onboarding flow (plan issue #76): the model's step machine
/// owns the navigation — the NavigationStack path mirrors it, so the system
/// back (swipe or button) walks the machine backwards and the drafts
/// survive because they live in the model. The OTP verify persists the
/// session and the reactive gate (slice 2) swaps this flow for home; no
/// navigation call happens here. The ritual and the contact moment (slice
/// 4) stack on top of the same machine.
struct OnboardingScreen: View {
    let model: OnboardingModel

    var body: some View {
        NavigationStack(path: pathBinding) {
            WelcomeView(onStart: model.startTapped)
                .navigationDestination(for: OnboardingStep.self) { step in
                    destinationView(for: step)
                }
        }
    }

    private var pathBinding: Binding<[OnboardingStep]> {
        Binding(
            get: { model.navigationPath() },
            set: { model.navigationPathChanged($0) }
        )
    }

    private func destinationView(for step: OnboardingStep) -> some View {
        Group {
            switch step {
            case .welcome:
                EmptyView() // the root; never pushed
            case .phone:
                OtpRequestView(model: model)
            case .code:
                OtpCodeView(model: model)
            }
        }
        .navigationBarTitleDisplayMode(.inline)
        .toolbarBackground(.hidden, for: .navigationBar)
    }
}

#Preview("Bienvenida") {
    OnboardingScreen(model: OnboardingModel(previewState: OnboardingUiState()))
}

#Preview("Código") {
    OnboardingScreen(model: OnboardingModel(previewState: OnboardingUiState(
        step: .code,
        phone: "+52 55 1234 5678",
        code: "48291"
    )))
}
