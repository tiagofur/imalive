import Foundation
@testable import Tranqui

/// The persistence seam without touching the Keychain: shared by the
/// session and refresher tests (the refresher suite uses it through the
/// real SessionStore, so the reactive gate is exercised end to end).
final class InMemoryPersisting: SessionPersisting, @unchecked Sendable {
    private let lock = NSLock()
    private var stored: StoredSession?
    private var onboardingComplete = false

    init(initial: StoredSession? = nil) {
        stored = initial
    }

    func load() -> StoredSession? {
        lock.withLock { stored }
    }

    func save(_ session: StoredSession) {
        lock.withLock { stored = session }
    }

    func clear() {
        lock.withLock {
            stored = nil
            onboardingComplete = false
        }
    }

    func setOnboardingComplete(_ complete: Bool) {
        lock.withLock { onboardingComplete = complete }
    }

    func isOnboardingComplete() -> Bool {
        lock.withLock { onboardingComplete }
    }
}
