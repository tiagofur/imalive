import Foundation
import Observation

/// Which surface owns the screen, decided by the persisted session (the
/// reactive gate of plan issue #76, mirror of Android's `HomeGate`).
enum HomeGate: Equatable, Sendable {
    /// The store has not answered yet.
    case loading

    /// No session, or the session exists but the onboarding has not
    /// finished (slice 4): the onboarding flow owns the screen.
    case needsOnboarding

    /// Session present and the onboarding finished: home, with the real
    /// streak.
    case ready
}

/// The seam the network layer consumes: the in-memory cache that the bearer
/// attach and the token refresher read on every call (never a per-request
/// Keychain hit).
protocol SessionStoring: Sendable {
    func cachedSession() async -> StoredSession?
    func save(_ session: StoredSession) async
    func clear() async
}

/// The reactive session state (plan issue #76): caches what
/// `KeychainSessionStore` persists and exposes the gate the app root
/// switches on. A session wiped by the refresher (a rejected refresh) walks
/// the user back to onboarding without manual navigation.
///
/// The gate needs session AND the finished-onboarding flag (slice 4, as on
/// Android): the OTP verify persists the session and the flow continues —
/// only the ritual and the trusted contact (or the honest skip) open home.
@MainActor
@Observable
final class SessionStore: SessionStoring {
    private(set) var session: StoredSession?
    private(set) var onboardingComplete = false
    private(set) var gate: HomeGate = .loading

    private let persisting: any SessionPersisting

    /// Restores the persisted session synchronously: the Keychain read is
    /// atomic, so the gate is already resolved when init returns.
    init(persisting: any SessionPersisting) {
        self.persisting = persisting
        let restored = persisting.load()
        session = restored
        onboardingComplete = persisting.isOnboardingComplete()
        gate = Self.gate(session: restored, onboardingComplete: onboardingComplete)
    }

    func save(_ newSession: StoredSession) {
        persisting.save(newSession)
        session = newSession
        gate = Self.gate(session: newSession, onboardingComplete: onboardingComplete)
    }

    /// Marks the onboarding as finished; the gate then lets the user into
    /// home (issue #76 slice 4). The flow calls it on the final step — the
    /// trusted contact sent, the free-plan copy or the honest skip.
    func completeOnboarding() {
        persisting.setOnboardingComplete(true)
        onboardingComplete = true
        gate = Self.gate(session: session, onboardingComplete: true)
    }

    /// Wipes the Keychain (and the flag with it) and the cache: used on
    /// logout and on a rejected refresh (the refresher calls this through
    /// the `SessionStoring` seam).
    func clear() {
        persisting.clear()
        session = nil
        onboardingComplete = false
        gate = Self.gate(session: nil, onboardingComplete: false)
    }

    // MARK: SessionStoring — called from the network actors. A token

    // rotation rewrites the cache the interceptors read; nothing reloads.

    func cachedSession() -> StoredSession? {
        session
    }

    private static func gate(session: StoredSession?, onboardingComplete: Bool) -> HomeGate {
        session != nil && onboardingComplete ? .ready : .needsOnboarding
    }
}
