import SwiftUI

/// Paso 2, fase teléfono (brief): "¿A qué teléfono te enviamos el código?"
/// — E.164 on the phone pad; the CTA héroe stays anchored at the bottom and
/// the errors speak the brief's copy, never the raw detail.
struct OtpRequestView: View {
    /// The step machine: the view reads its state and answers through it.
    let model: OnboardingModel

    var body: some View {
        VStack(alignment: .leading, spacing: 0) {
            Text(String(localized: "onboarding.phone_title"))
                .font(DesignTokens.headline)
            Text(String(localized: "onboarding.phone_sub"))
                .font(DesignTokens.body)
                .foregroundStyle(.secondary)
                .padding(.top, 8)
            TextField(
                String(localized: "onboarding.phone_field_hint"),
                text: Binding(
                    get: { model.state.phone },
                    set: { model.phoneChanged($0) }
                )
            )
            .font(DesignTokens.body)
            .keyboardType(.phonePad)
            .textInputAutocapitalization(.never)
            .autocorrectionDisabled()
            .submitLabel(.done)
            .onSubmit(model.requestCodeTapped)
            .accessibilityLabel(String(localized: "onboarding.phone_field_label"))
            .padding(.vertical, 12)
            .background(
                RoundedRectangle(cornerRadius: 8)
                    .strokeBorder(
                        model.state.phoneError == nil ? Color.secondary.opacity(0.4) : DesignTokens.critical,
                        lineWidth: 1
                    )
            )
            .padding(.top, 24)
            if let error = model.state.phoneError {
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
                    title: String(localized: "onboarding.phone_cta"),
                    isEnabled: model.state.busy == nil && model.state.rateLimitSeconds == 0,
                    action: model.requestCodeTapped
                )
                if model.state.busy == .sendingCode {
                    BusyRow(label: String(localized: "onboarding.busy_sending_code"))
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

    /// The stable Problem.code maps to the brief's copy; the raw detail
    /// never reaches the user.
    private static func copy(for error: PhoneError) -> String {
        switch error {
        case .invalid: String(localized: "onboarding.error_phone_invalid")
        case .offline: String(localized: "onboarding.error_offline")
        case .retry: String(localized: "onboarding.error_retry")
        }
    }
}

#Preview("Vacío") {
    OtpRequestView(model: OnboardingModel(previewState: OnboardingUiState(step: .phone)))
}

#Preview("Teléfono inválido") {
    OtpRequestView(model: OnboardingModel(previewState: OnboardingUiState(
        step: .phone,
        phone: "55 1234",
        phoneError: .invalid
    )))
}

#Preview("Sin conexión") {
    OtpRequestView(model: OnboardingModel(previewState: OnboardingUiState(
        step: .phone,
        phone: "+5215512345678",
        phoneError: .offline
    )))
}

#Preview("Rate limit") {
    OtpRequestView(model: OnboardingModel(previewState: OnboardingUiState(
        step: .phone,
        phone: "+5215512345678",
        rateLimitSeconds: 47
    )))
}

#Preview("Enviando") {
    OtpRequestView(model: OnboardingModel(previewState: OnboardingUiState(
        step: .phone,
        phone: "+5215512345678",
        busy: .sendingCode
    )))
}
