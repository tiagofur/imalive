import SwiftUI

/// Paso 2, fase código (brief): six digits with SMS autofill, auto-verify
/// at the sixth, the "Entrar" backup and the resend with the live cooldown
/// fed by Retry-After.
struct OtpCodeView: View {
    let model: OnboardingModel

    var body: some View {
        VStack(alignment: .leading, spacing: 0) {
            Text(String(localized: "onboarding.code_title"))
                .font(DesignTokens.headline)
            Text(String(format: String(localized: "onboarding.code_sub"), model.state.phone))
                .font(DesignTokens.body)
                .foregroundStyle(.secondary)
                .padding(.top, 8)
            OtpInput(
                value: model.state.code,
                isError: model.state.codeError != nil,
                onChange: model.codeChanged
            )
            .padding(.top, 24)
            if let error = model.state.codeError {
                Text(Self.copy(for: error))
                    .font(DesignTokens.label)
                    .foregroundStyle(DesignTokens.critical)
                    .padding(.top, 12)
            }
            Spacer()
            VStack(spacing: 8) {
                if model.state.rateLimitSeconds > 0 {
                    rateLimitCopy
                }
                HeroButton(
                    title: String(localized: "onboarding.code_enter"),
                    isEnabled: model.state.busy == nil && model.state.code.count == Self.codeLength,
                    action: model.verifyTapped
                )
                if let busy = model.state.busy {
                    // Both long operations speak: the resend also runs from
                    // this phase, and its wait copy is "Enviando tu código…".
                    BusyRow(label: Self.busyCopy(for: busy))
                } else {
                    Button(String(localized: "onboarding.code_resend")) {
                        model.resendTapped()
                    }
                    .font(DesignTokens.label)
                    .disabled(model.state.rateLimitSeconds > 0)
                    .frame(maxWidth: .infinity)
                    .padding(.top, 4)
                }
            }
            .padding(.bottom, 24)
        }
        .padding(.horizontal, 24)
    }

    private var rateLimitCopy: some View {
        Text(String(
            format: String(localized: "onboarding.error_rate_limited"),
            model.state.rateLimitSeconds
        ))
        .font(DesignTokens.label)
        .foregroundStyle(DesignTokens.critical)
    }

    private static let codeLength = 6

    private static func busyCopy(for busy: OnboardingBusy) -> String {
        switch busy {
        case .sendingCode: String(localized: "onboarding.busy_sending_code")
        case .verifying: String(localized: "onboarding.busy_verifying")
        case .savingRitual, .savingContact:
            // The OTP screen never shows the ritual's busy states; they
            // exist in the same enum, so the switch must know them.
            String(localized: "onboarding.busy_sending_code")
        }
    }

    /// The stable Problem.code maps to the brief's copy; the raw detail
    /// never reaches the user.
    private static func copy(for error: CodeError) -> String {
        switch error {
        case .invalid: String(localized: "onboarding.error_otp_invalid")
        case .expired: String(localized: "onboarding.error_otp_expired")
        case .offline: String(localized: "onboarding.error_offline")
        case .retry: String(localized: "onboarding.error_retry")
        }
    }
}

#Preview("Vacío") {
    OtpCodeView(model: OnboardingModel(previewState: OnboardingUiState(
        step: .code,
        phone: "+52 55 1234 5678"
    )))
}

#Preview("Código inválido") {
    OtpCodeView(model: OnboardingModel(previewState: OnboardingUiState(
        step: .code,
        phone: "+52 55 1234 5678",
        code: "111111",
        codeError: .invalid
    )))
}

#Preview("Caducado, reenviado") {
    OtpCodeView(model: OnboardingModel(previewState: OnboardingUiState(
        step: .code,
        phone: "+52 55 1234 5678",
        codeError: .expired
    )))
}

#Preview("Verificando") {
    OtpCodeView(model: OnboardingModel(previewState: OnboardingUiState(
        step: .code,
        phone: "+52 55 1234 5678",
        code: "482913",
        busy: .verifying
    )))
}

#Preview("Rate limit") {
    OtpCodeView(model: OnboardingModel(previewState: OnboardingUiState(
        step: .code,
        phone: "+52 55 1234 5678",
        rateLimitSeconds: 47
    )))
}
