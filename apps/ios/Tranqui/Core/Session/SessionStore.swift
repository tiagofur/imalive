import Foundation
import Observation

/// Which surface owns the screen, decided by the persisted session (the
/// reactive gate of plan issue #76, mirror of Android's `HomeGate`).
enum HomeGate: Equatable, Sendable {
    /// The store has not answered yet.
    case loading

    /// No session: the onboarding flow owns the screen.
    case needsOnboarding

    /// Session present: home, with the real streak.
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
/// The gate keys on session presence for now; the onboarding-finished flag
/// joins with the onboarding flow itself (slices 3-4), as on Android.
@MainActor
@Observable
final class SessionStore: SessionStoring {
    private(set) var session: StoredSession?
    private(set) var gate: HomeGate = .loading

    private let persisting: any SessionPersisting

    /// Restores the persisted session synchronously: the Keychain read is
    /// atomic, so the gate is already resolved when init returns.
    init(persisting: any SessionPersisting) {
        self.persisting = persisting
        let restored = persisting.load()
        session = restored
        gate = Self.gate(for: restored)
    }

    func save(_ newSession: StoredSession) {
        persisting.save(newSession)
        session = newSession
        gate = Self.gate(for: newSession)
    }

    /// Wipes the Keychain and the cache: used on logout and on a rejected
    /// refresh (the refresher calls this through the `SessionStoring` seam).
    func clear() {
        persisting.clear()
        session = nil
        gate = Self.gate(for: nil)
    }

    // MARK: SessionStoring — called from the network actors. A token

    // rotation rewrites the cache the interceptors read; nothing reloads.

    func cachedSession() -> StoredSession? {
        session
    }

    private static func gate(for session: StoredSession?) -> HomeGate {
        session == nil ? .needsOnboarding : .ready
    }
}
