import SwiftUI

/// Paso 1 (brief, "la promesa"): the emotional promise, not features. One
/// protagonist CTA anchored at the bottom; the honest disclaimer speaks in
/// a low voice under it — the first true thing the product says about
/// itself (voz baja, nunca un modal).
struct WelcomeView: View {
    let onStart: () -> Void

    var body: some View {
        VStack(alignment: .leading, spacing: 0) {
            Spacer(minLength: 96)
            Text(String(localized: "onboarding.welcome_title"))
                .font(DesignTokens.display)
            Text(String(localized: "onboarding.welcome_sub"))
                .font(DesignTokens.body)
                .foregroundStyle(.secondary)
                .padding(.top, 16)
            Spacer()
            VStack(spacing: 16) {
                HeroButton(title: String(localized: "onboarding.welcome_cta"), action: onStart)
                Text(String(format: String(localized: "onboarding.welcome_disclaimer"), Bundle.main.productDisplayName))
                    .font(DesignTokens.label)
                    .foregroundStyle(.secondary)
                    .multilineTextAlignment(.center)
                    .padding(.bottom, 24)
                    .frame(maxWidth: .infinity)
            }
        }
        .padding(.horizontal, 24)
    }
}

#Preview {
    WelcomeView(onStart: {})
}
