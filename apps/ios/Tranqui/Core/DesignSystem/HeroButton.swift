import SwiftUI

/// The hero CTA — the OkButton DNA (design-system.md): one protagonist
/// action per screen, pill radius, 56 pt tall (targets ≥ 44 pt). Home and
/// onboarding share it.
struct HeroButton: View {
    let title: String
    var isEnabled = true
    let action: () -> Void

    var body: some View {
        Button(action: action) {
            Text(title)
                .font(DesignTokens.title)
                .frame(maxWidth: .infinity, minHeight: DesignTokens.heroHeight)
        }
        .buttonStyle(.borderedProminent)
        .buttonBorderShape(.capsule)
        .tint(DesignTokens.primary)
        .disabled(!isEnabled)
    }
}

#Preview {
    VStack(spacing: 16) {
        HeroButton(title: String(localized: "onboarding.welcome_cta"), action: {})
        HeroButton(title: String(localized: "onboarding.phone_cta"), isEnabled: false, action: {})
    }
    .padding(24)
}
