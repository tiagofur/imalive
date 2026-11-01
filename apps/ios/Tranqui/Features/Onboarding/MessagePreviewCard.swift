import SwiftUI

/// The memorable moment (brief paso 4): a chat bubble showing, with the real
/// names the user is typing, the exact message the trusted person will
/// receive. Mirror, not promise — honesty is the feature (issue #76 slice 4,
/// the iOS twin of Android's MessagePreviewCard).
struct MessagePreviewCard: View {
    let inviterName: String
    let contactName: String

    var body: some View {
        let message = welcomePreview(
            template: WelcomeTemplate(
                text: String(localized: "onboarding.contact.welcome_message"),
                inviterFallback: String(localized: "onboarding.contact.preview_you"),
                contactFallback: String(localized: "onboarding.contact.preview_them")
            ),
            inviterName: inviterName,
            contactName: contactName,
            appName: Bundle.main.productDisplayName
        )
        // A received bubble: the small corner opens toward the sender.
        Text(message)
            .font(.body)
            .padding(16)
            .frame(maxWidth: .infinity, alignment: .leading)
            .background(Color(.systemGray6))
            .clipShape(
                UnevenRoundedRectangle(
                    topLeadingRadius: 4,
                    bottomLeadingRadius: 16,
                    bottomTrailingRadius: 16,
                    topTrailingRadius: 16,
                    style: .continuous
                )
            )
            .accessibilityElement(children: .combine)
            .accessibilityLabel(message)
    }
}
