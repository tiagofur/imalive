import SwiftUI

/// The ritual progress of the brief ("línea fina de 4 segmentos arriba"):
/// not bureaucracy — no big numbers, the accessible name carries the count
/// ("Paso N de 4"). Bienvenida is segment 1; the two OTP phases share
/// segment 2 (the ritual and the contact moment join in slice 4).
struct OnboardingProgress: View {
    /// 1-based; the four segments of the brief.
    let currentSegment: Int

    var body: some View {
        HStack(spacing: 6) {
            ForEach(0 ..< Self.totalSegments, id: \.self) { segment in
                Capsule()
                    .fill(segment < currentSegment ? DesignTokens.primary : Color.secondary.opacity(0.25))
                    .frame(height: 3)
            }
        }
        .padding(.horizontal, 24)
        .padding(.vertical, 16)
        .accessibilityElement(children: .ignore)
        .accessibilityLabel(String(format: String(localized: "onboarding.progress_a11y"), currentSegment))
    }

    private static let totalSegments = 4
}

#Preview {
    OnboardingProgress(currentSegment: 2)
}
