import Foundation
import Observation

/// The onboarding step machine (plan issue #76, the iOS mirror of Android's
/// OnboardingViewModel): every user intent is a small named function, and
/// the contract's stable Problem.codes map to human states — the raw detail
/// never reaches the screen (brief §6).
@MainActor
@Observable
final class OnboardingModel {
    /// Internal-set on purpose: the ritual extension (same module) writes it.
    internal(set) var state = OnboardingUiState()

    private let auth: any AuthRepository
    // The ritual extension (OnboardingRitual.swift, same module) writes the
    // state and reads these: internal on purpose, views never see them.
    let profiles: any ProfileRepository
    let contacts: any ContactsRepository
    let sessions: SessionStore
    private var cooldownTask: Task<Void, Never>?

    /// The cooldown tick: 1 s in production. The tests inject a long tick to
    /// freeze the countdown at an exact value and a short one to watch it
    /// drain — the same trick as Android's virtual test dispatcher.
    var cooldownTick: Duration = .seconds(1)

    init(
        auth: any AuthRepository,
        profiles: any ProfileRepository,
        contacts: any ContactsRepository,
        sessions: SessionStore
    ) {
        self.auth = auth
        self.profiles = profiles
        self.contacts = contacts
        self.sessions = sessions
        // A process death mid-onboarding (the OTP verify already saved the
        // session) resumes at the ritual: the auth phases are done.
        if sessions.session != nil {
            state.step = .ritual
        }
    }

    // MARK: - Intents

    func startTapped() {
        state.step = .phone
        state.phoneError = nil
    }

    func phoneChanged(_ raw: String) {
        // The field accepts whatever the keyboard inserts; only ASCII digits
        // and the leading + reach the draft (E.164 discipline).
        state.phone = raw.filter { Self.digits.contains($0) || $0 == "+" }
        state.phoneError = nil
    }

    func requestCodeTapped() {
        guard state.busy == nil, state.rateLimitSeconds == 0 else { return }
        guard (try? Self.e164.wholeMatch(in: state.phone)) != nil else {
            state.phoneError = .invalid
            return
        }
        sendCode(markExpiredOnSuccess: false)
    }

    func resendTapped() {
        guard state.busy == nil, state.rateLimitSeconds == 0 else { return }
        sendCode(markExpiredOnSuccess: false)
    }

    func codeChanged(_ raw: String) {
        state.code = String(raw.filter { Self.digits.contains($0) }.prefix(Self.codeLength))
        state.codeError = nil
        // Auto-verify at the sixth digit (brief paso 2); the "Entrar" button
        // is the backup for when the SMS autofill never arrives.
        if state.code.count == Self.codeLength {
            verify()
        }
    }

    func verifyTapped() {
        guard state.busy == nil else { return }
        verify()
    }

    /// The NavigationStack path mirrors the machine: the system back pops it
    /// and the drafts follow the machine's rule — the phone survives, a
    /// half-written code does not (a stale code can only fail). Forward
    /// navigation never comes from here; only the intents push steps.
    func navigationPathChanged(_ path: [OnboardingStep]) {
        let target = path.last ?? .welcome
        guard target != state.step else { return }
        switch target {
        case .welcome:
            state.step = .welcome
        case .phone:
            state.code = ""
            state.codeError = nil
            state.step = .phone
        case .code:
            break
        case .ritual:
            state.step = .ritual
        case .contact:
            state.step = .contact
        }
    }

    /// The path the NavigationStack renders for the current step (welcome is
    /// the root; phone and code stack on top of it). The ritual and contact
    /// steps are flat (the system back is disabled there on purpose: leaving
    /// mid-ritual with a saved session would strand a half-open account).
    func navigationPath() -> [OnboardingStep] {
        switch state.step {
        case .welcome: []
        case .phone: [.phone]
        case .code: [.phone, .code]
        case .ritual, .contact: []
        }
    }

    // MARK: - Wire

    private func sendCode(markExpiredOnSuccess: Bool) {
        let phone = state.phone
        state.busy = .sendingCode
        state.phoneError = nil
        state.codeError = nil
        Task {
            do {
                try await auth.requestOtp(phone: phone)
                state.busy = nil
                state.step = .code
                // The brief's expiry flow: when the fresh code of the
                // auto-reenvío lands, the copy says so.
                state.codeError = markExpiredOnSuccess ? .expired : nil
            } catch let error as ApiError {
                handle(error, fallback: {
                    state.busy = nil
                    state.phoneError = .retry
                })
            } catch is URLError {
                state.busy = nil
                state.phoneError = .offline
            } catch {
                state.busy = nil
                state.phoneError = .retry
            }
        }
    }

    private func verify() {
        guard state.busy == nil, state.code.count == Self.codeLength else { return }
        let phone = state.phone
        let code = state.code
        state.busy = .verifying
        state.codeError = nil
        Task {
            do {
                try await auth.verifyOtp(phone: phone, code: code)
                // The repo persisted the session: the flow continues at the
                // ritual (slice 4 — a session alone is not home). The reset
                // keeps a session wiped later (a rejected refresh) reopening
                // the flow fresh instead of mid-code.
                state = OnboardingUiState()
                state.step = .ritual
            } catch let error as ApiError {
                handle(error, fallback: {
                    state.busy = nil
                    state.codeError = .retry
                })
            } catch is URLError {
                state.busy = nil
                state.codeError = .offline
            } catch {
                state.busy = nil
                state.codeError = .retry
            }
        }
    }

    /// The brief: the app switches on the stable Problem.code and never
    /// shows the raw detail. The fallback belongs to the phase in flight —
    /// a non-mapped code during the request must not write into the code
    /// slot, where the user can't see it.
    private func handle(_ error: ApiError, fallback: () -> Void) {
        switch error.code {
        case .phoneInvalid:
            state.busy = nil
            state.phoneError = .invalid
        case .otpInvalid:
            // The written code stays for rewrite (brief paso 2); the cells
            // carry the error because the iOS 17 TextField cannot select.
            state.busy = nil
            state.codeError = .invalid
        case .otpExpired:
            // Auto-reenvío (brief paso 2): a fresh code goes out; when it
            // lands, the copy confirms it.
            sendCode(markExpiredOnSuccess: true)
        case .rateLimited:
            state.busy = nil
            // Real countdown from the Retry-After header (60 s default when
            // the header is missing or invalid, plan issue #76): it blocks
            // the resend while live.
            startCooldown(seconds: error.retryAfterSeconds ?? Self.defaultCooldownSeconds)
        case .unauthorized, .validationFailed, .contactLimitReached, .contactNotFound, .sosPaused, .unknown:
            fallback()
        }
    }

    private func startCooldown(seconds: Int) {
        cooldownTask?.cancel()
        state.rateLimitSeconds = seconds
        cooldownTask = Task {
            var remaining = seconds
            while remaining > 0 {
                do {
                    try await Task.sleep(for: cooldownTick)
                } catch {
                    return // cancelled: a fresh countdown took over
                }
                remaining -= 1
                state.rateLimitSeconds = remaining
            }
        }
    }

    private static let codeLength = 6
    private static let defaultCooldownSeconds = 60
    static let digits = "0123456789"
    static let maxNameLength = 80

    /// E.164: + followed by 8–15 digits, country code not starting 0.
    static let e164 = /^\+[1-9]\d{7,14}$/

    /// Minutes-from-midnight → the contract's HH:MM (24h).
    static func hhmm(_ minutes: Int) -> String {
        String(format: "%02d:%02d", minutes / 60, minutes % 60)
    }
}

#if DEBUG
    extension OnboardingModel {
        /// Previews and tests only: jumps the machine to a step without the
        /// wire (the ritual suites exercise contact flows directly).
        func jumpTo(_ step: OnboardingStep) {
            state.step = step
        }

        /// Previews only: a model frozen at a given state (the preview seams
        /// answer nothing).
        convenience init(previewState: OnboardingUiState) {
            self.init(
                auth: PreviewAuthRepository(),
                profiles: PreviewSupport.profiles,
                contacts: PreviewSupport.contacts,
                sessions: PreviewSupport.sessions
            )
            state = previewState
        }
    }
#endif
