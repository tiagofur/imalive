import Foundation

/// In-memory persistence for the smoke mode (`-tranqui-session-stub`) and
/// the Xcode Previews: process-local, nothing reaches the Keychain or the
/// real defaults — test tokens are not real sessions.
final class EphemeralSessionPersisting: SessionPersisting, @unchecked Sendable {
    private let lock = NSLock()
    private var stored: StoredSession?
    private var onboardingComplete = false

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
