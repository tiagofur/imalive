import SwiftUI

/// The short wait copy of the brief ("spinner corto, sin animaciones de
/// espera largas"): one line, small spinner, human words.
struct BusyRow: View {
    let label: String

    var body: some View {
        HStack(spacing: 8) {
            ProgressView()
                .controlSize(.small)
            Text(label)
                .font(DesignTokens.label)
                .foregroundStyle(.secondary)
        }
        .frame(maxWidth: .infinity)
        .padding(.top, 8)
        .accessibilityElement(children: .combine)
    }
}
