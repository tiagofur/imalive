import Foundation

/// The mirror of the server's welcome (R-03): `template` is the exact string
/// `welcomeMessageV1` renders in backend/internal/circle/http.go,
/// interpolated the same way — `%1$@` is the greeting ("Hola {contacto}",
/// hardcoded in the server too), `%2$@` the inviter, `%3$@` the app. Three
/// guards pin it: the copy guard pins the resource, the pure function below
/// is unit-tested against the server-rendered sentence, and the composable
/// test asserts the bubble. Follow-up: an additive endpoint that hands the
/// app the template and deletes the mirror (issue #76 closure note).
struct WelcomeTemplate: Equatable, Sendable {
    /// The exact string the server renders (see `welcomePreview`).
    let text: String
    /// Neutral placeholders of the empty preview ("[tu nombre]"); the real
    /// send never carries blank names (the API requires 1..80 characters).
    let inviterFallback: String
    let contactFallback: String
}

/// Interpolates the server's welcome exactly like
/// `welcomeMessageV1(inviter, contact, app)` does (issue #76 slice 4).
func welcomePreview(
    template: WelcomeTemplate,
    inviterName: String,
    contactName: String,
    appName: String
) -> String {
    let inviter = inviterName.isBlank ? template.inviterFallback : inviterName
    let contact = contactName.isBlank ? template.contactFallback : contactName
    return String(format: template.text, "Hola \(contact)", inviter, appName)
}

extension String {
    var isBlank: Bool {
        trimmingCharacters(in: .whitespacesAndNewlines).isEmpty
    }

    /// nil when blank: guards read as bindings instead of negations.
    var nonEmpty: String? {
        let trimmed = trimmingCharacters(in: .whitespacesAndNewlines)
        return trimmed.isEmpty ? nil : trimmed
    }
}
