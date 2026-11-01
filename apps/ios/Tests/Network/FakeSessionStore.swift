import Foundation
@testable import Tranqui

/// In-memory SessionStoring for the network tests: the same contract the
/// Keychain-backed SessionStore honors, without disk or the MainActor. The
/// refresher tests use the real SessionStore instead — this fake is for the
/// transport tests.
final class FakeSessionStore: SessionStoring, @unchecked Sendable {
    private let lock = NSLock()
    private var stored: StoredSession?
    private var clears = 0

    /// Seeds the cache before the call under test (sync: pre-arrange only).
    func store(_ session: StoredSession) {
        lock.withLock { stored = session }
    }

    func cachedSession() async -> StoredSession? {
        lock.withLock { stored }
    }

    func save(_ session: StoredSession) async {
        lock.withLock { stored = session }
    }

    func clear() async {
        lock.withLock {
            stored = nil
            clears += 1
        }
    }

    var clearCount: Int {
        lock.withLock { clears }
    }

    var currentSession: StoredSession? {
        lock.withLock { stored }
    }
}
