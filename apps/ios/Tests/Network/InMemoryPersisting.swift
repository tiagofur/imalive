import Foundation
@testable import Tranqui

/// The persistence seam without touching the Keychain: shared by the
/// session and refresher tests (the refresher suite uses it through the
/// real SessionStore, so the reactive gate is exercised end to end).
final class InMemoryPersisting: SessionPersisting, @unchecked Sendable {
    private let lock = NSLock()
    private var stored: StoredSession?

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
        lock.withLock { stored = nil }
    }
}
