import SwiftUI

/// Root of the onboarding flow (plan issue #76): the model's step machine
/// owns the navigation — the NavigationStack path mirrors it, so the system
/// back (swipe or button) walks the machine backwards and the drafts
/// survive because they live in the model. The ritual progress line rides
/// on top of every step. The OTP verify persists the session and the
/// reactive gate (slice 2) swaps this flow for home; no navigation call
/// happens here. The ritual and the contact moment (slice 4) stack on top
/// of the same machine.
struct OnboardingScreen: View {
    let model: OnboardingModel

    var body: some View {
        NavigationStack(path: pathBinding) {
            stepView(step: .welcome) {
                WelcomeView(onStart: model.startTapped)
            }
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

    @ViewBuilder
    private func destinationView(for step: OnboardingStep) -> some View {
        switch step {
        case .welcome:
            EmptyView() // the root; never pushed
        case .phone:
            stepView(step: step) {
                OtpRequestView(model: model)
            }
        case .code:
            stepView(step: step) {
                OtpCodeView(model: model)
            }
        }
    }

    /// One step = the ritual progress line on top (brief: "línea fina de 4
    /// segmentos arriba") and the step's single column below it.
    private func stepView(step: OnboardingStep, @ViewBuilder content: () -> some View) -> some View {
        VStack(spacing: 0) {
            OnboardingProgress(currentSegment: step.progressSegment)
            content()
        }
        .navigationBarTitleDisplayMode(.inline)
        .toolbarBackground(.hidden, for: .navigationBar)
    }
}

private extension OnboardingStep {
    /// Welcome is segment 1; the two OTP phases share segment 2 (brief: 4
    /// visible steps; phone and code are step 2). The ritual and the
    /// contact moment join in slice 4.
    var progressSegment: Int {
        switch self {
        case .welcome: 1
        case .phone, .code: 2
        }
    }
}

#Preview("Bienvenida") {
    OnboardingScreen(model: OnboardingModel(previewState: OnboardingUiState()))
}

#Preview("Teléfono") {
    OnboardingScreen(model: OnboardingModel(previewState: OnboardingUiState(
        step: .phone,
        phone: "+52 55 1234 5678"
    )))
}

#Preview("Código") {
    OnboardingScreen(model: OnboardingModel(previewState: OnboardingUiState(
        step: .code,
        phone: "+52 55 1234 5678",
        code: "48291"
    )))
}
