import Foundation

/// The linear steps of the onboarding (plan issue #76): no custom
/// navigation — the machine owns the step (the NavigationStack path mirrors
/// it, so the system back walks it backwards) and the phone/code drafts
/// survive because they live here. Slice 3 lands the auth steps; the ritual
/// and the contact moment join in slice 4. The verify persists the session
/// and the reactive gate (slice 2) lands home — leaving the flow is never a
/// step of the machine.
enum OnboardingStep: Equatable, Hashable, Sendable {
    case welcome
    case phone
    case code
    /// Paso 3, the ritual (slice 4): the user's name and the reminder time.
    case ritual
    /// Paso 4, the memorable moment (slice 4): the trusted contact with the
    /// message mirror.
    case contact
}

/// Which long operation is running: the copy differs per phase (brief paso
/// 2: "Enviando tu código…" / "Verificando…").
enum OnboardingBusy: Equatable, Sendable {
    case sendingCode
    case verifying
    case savingRitual
    case savingContact
}

enum PhoneError: Equatable, Sendable {
    /// The E.164 check (local or the server's PHONE_INVALID) failed.
    case invalid

    /// No network: the draft stays, the user retries.
    case offline

    /// An unexpected business answer: human copy, never the raw detail.
    case retry
}

enum RitualError: Equatable, Sendable {
    /// The name is blank: the welcome message needs someone to call by.
    case invalidName
    case offline
    case retry
}

enum ContactError: Equatable, Sendable {
    /// The E.164 check failed (local or the server's PHONE_INVALID).
    case invalidPhone
    case offline
    case retry
}

enum CodeError: Equatable, Sendable {
    /// OTP_INVALID: the written code stays for rewrite (brief paso 2).
    case invalid

    /// OTP_EXPIRED: a fresh code went out automatically (the auto-reenvío).
    case expired

    case offline
    case retry
}

/// The one screen-state the onboarding renders. The model writes it, the
/// views read it; nothing mutates it behind the machine's intents.
struct OnboardingUiState: Equatable, Sendable {
    var step: OnboardingStep = .welcome
    var phone = ""
    var code = ""
    var busy: OnboardingBusy?
    var phoneError: PhoneError?
    var codeError: CodeError?

    /// Live resend countdown in seconds: > 0 disables "Enviarlo de nuevo"
    /// and shows the rate-limit copy with the seconds (brief paso 2). Fed
    /// by the Retry-After header.
    var rateLimitSeconds = 0

    /// The ritual (paso 3).
    var name = ""
    /// The reminder time as minutes from midnight (default 21:00, the
    /// product default and the server's column default).
    var checkinMinutes = 21 * 60
    var ritualError: RitualError?

    // The contact (paso 4, the memorable moment).
    var contactName = ""
    var contactPhone = ""
    var contactError: ContactError?
    /// CONTACT_LIMIT_REACHED is not an error: the free plan keeps 1 contact
    /// and the copy says when to add more (brief paso 4, honesto).
    var quotaNotice = false
    /// The saved contact's display name: the arrival copy carries it
    /// ("Listo. María ya sabe que desde hoy cuentas con ella.").
    var savedContactName: String?
}
