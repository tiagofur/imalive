import SwiftUI

/// The six OTP cells of the brief (paso 2): one real TextField carries the
/// input — invisible under the drawn cells — so the SMS autofill
/// (.textContentType(.oneTimeCode)), paste and the keyboard all land in a
/// single field. The model owns the digits; this piece is stateless. On
/// error the written code stays for rewrite (the model keeps it); the
/// programmatic "select all" of the brief needs the iOS 18 TextField
/// selection API — above the iOS 17 deployment target — so the cells carry
/// the error instead.
struct OtpInput: View {
    let value: String
    var isError = false
    let onChange: (String) -> Void

    var body: some View {
        TextField(
            "",
            text: Binding(
                get: { value },
                set: { onChange($0) }
            )
        )
        .textContentType(.oneTimeCode)
        .keyboardType(.numberPad)
        .tint(.clear)
        .foregroundStyle(.clear)
        .frame(height: 56)
        .accessibilityLabel(String(localized: "onboarding.code_field_a11y"))
        .overlay {
            HStack(spacing: 8) {
                ForEach(0 ..< Self.length, id: \.self) { index in
                    OtpCell(
                        character: value.character(at: index),
                        state: cellState(at: index)
                    )
                }
            }
            .allowsHitTesting(false)
        }
    }

    private func cellState(at index: Int) -> OtpCell.State {
        if isError {
            return .error
        }
        if index == value.count {
            return .active
        }
        return .idle
    }

    private static let length = 6
}

private extension String {
    /// The digit drawn in the cell at `index`, if any.
    func character(at index: Int) -> Character? {
        guard index < count else { return nil }
        return self[self.index(startIndex, offsetBy: index)]
    }
}

private struct OtpCell: View {
    enum State {
        case idle, active, error
    }

    let character: Character?
    let state: State

    var body: some View {
        ZStack {
            RoundedRectangle(cornerRadius: 8)
                .fill(.background)
            RoundedRectangle(cornerRadius: 8)
                .strokeBorder(borderColor, lineWidth: 1)
            Text(character.map(String.init) ?? "")
                .font(DesignTokens.headline)
                .minimumScaleFactor(0.5)
                .lineLimit(1)
        }
        .frame(height: 56)
    }

    private var borderColor: Color {
        switch state {
        case .error: DesignTokens.critical
        case .active: DesignTokens.primary
        case .idle: Color.secondary.opacity(0.4)
        }
    }
}

#Preview("Vacío") {
    OtpInput(value: "", onChange: { _ in })
        .padding(24)
}

#Preview("Escribiendo") {
    OtpInput(value: "48291", onChange: { _ in })
        .padding(24)
}

#Preview("Error") {
    OtpInput(value: "111111", isError: true, onChange: { _ in })
        .padding(24)
}
