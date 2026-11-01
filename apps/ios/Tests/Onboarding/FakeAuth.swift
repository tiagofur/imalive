import Foundation
@testable import Tranqui

/// The auth seam the onboarding model tests control: each call's failure is
/// programmable, the calls are counted, and a request can be held open until
/// the test releases it (the busy-guard test). The wire behavior is covered
/// by TranquiAPITests; the repo's session persistence by
/// NetworkRepositoriesTests.
final class FakeAuth: AuthRepository, @unchecked Sendable {
    private let lock = NSLock()
    private var requestCalls = 0
    private var verifyCalls = 0
    private var lastVerify: (phone: String, code: String)?
    private var held: CheckedContinuation<Void, Never>?

    /// When true, the next requestOtp suspends until release() — the busy
    /// machine the extra-taps test needs.
    var holdNextRequest = false
    var requestError: (any Error)?
    var verifyError: (any Error)?

    var requestCount: Int {
        lock.withLock { requestCalls }
    }

    var verifyCount: Int {
        lock.withLock { verifyCalls }
    }

    var verifiedPhone: String? {
        lock.withLock { lastVerify?.phone }
    }

    var verifiedCode: String? {
        lock.withLock { lastVerify?.code }
    }

    func requestOtp(phone _: String) async throws {
        lock.withLock { requestCalls += 1 }
        let hold = lock.withLock { holdNextRequest }
        if hold {
            lock.withLock { holdNextRequest = false }
            await withCheckedContinuation { continuation in
                lock.withLock { held = continuation }
            }
        }
        if let error = lock.withLock({ requestError }) {
            throw error
        }
    }

    /// Releases a request held by holdNextRequest.
    func release() {
        lock.withLock { held }?.resume()
        lock.withLock { held = nil }
    }

    func verifyOtp(phone: String, code: String) async throws {
        lock.withLock { verifyCalls += 1 }
        lock.withLock { lastVerify = (phone, code) }
        if let error = lock.withLock({ verifyError }) {
            throw error
        }
    }
}
