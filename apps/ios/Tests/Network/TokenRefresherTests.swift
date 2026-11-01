import Foundation
import Testing
@testable import Tranqui

// The refresh single-flight of #76 (the iOS mirror of Android's
// TokenAuthenticatorTest): concurrent 401s must produce exactly ONE
// POST /auth/refresh with real concurrency (TaskGroup); a rejected refresh
// must wipe the session (and the reactive gate must land on onboarding);
// an unreachable refresh must NOT (offline never logs the user out). The
// stack mirrors production: ONE API over ONE transport, the refresh call
// included, with the /auth/ path guard preventing recursion.
@MainActor
@Suite("TokenRefresher")
struct TokenRefresherTests {
    private let persisting = InMemoryPersisting()
    private let api: TranquiAPI
    private let store: SessionStore

    init() {
        store = SessionStore(persisting: persisting)
        let configuration = URLSessionConfiguration.ephemeral
        configuration.protocolClasses = [StubURLProtocol.self]
        api = TranquiAPI(
            baseURL: URL(string: "https://stub.local/v1")!,
            sessions: store,
            transport: URLSession(configuration: configuration)
        )
    }

    /// Counter the handler bumps: the heart of the suite is what it counts.
    private final class Counter: @unchecked Sendable {
        private let lock = NSLock()
        private var value = 0

        func increment() {
            lock.withLock { value += 1 }
        }

        var current: Int {
            lock.withLock { value }
        }
    }

    private func seedSession() {
        store.save(StoredSession(
            accessToken: "stale-access-token",
            refreshToken: "refresh-token-1",
            expiresAt: Date(timeIntervalSince1970: 0)
        ))
    }

    /// Dispatches by path: the profile answers 401 until the rotation lands,
    /// then the contract's profile example.
    private func stub401UntilRefresh(refreshes: Counter, refreshAnswer: @Sendable @escaping () -> StubResponse) {
        StubURLProtocol.reset { request in
            let path = request.url?.path ?? ""
            if path.hasSuffix("/auth/refresh") {
                refreshes.increment()
                return refreshAnswer()
            }
            if path.hasSuffix("/me") {
                if refreshes.current == 0 {
                    return StubResponse(
                        status: 401,
                        headers: ["Content-Type": "application/problem+json"],
                        body: Data(Fixtures.unauthorizedProblem.utf8)
                    )
                }
                return StubResponse(status: 200, body: Data(Fixtures.profile.utf8))
            }
            throw URLError(.unsupportedURL)
        }
    }

    private nonisolated static func rejectedRefreshAnswer() -> StubResponse {
        StubResponse(
            status: 401,
            headers: ["Content-Type": "application/problem+json"],
            body: Data(Fixtures.unauthorizedProblem.utf8)
        )
    }

    private nonisolated static func rotatedTokensAnswer() -> StubResponse {
        StubResponse(status: 200, body: Data(Fixtures.tokens.utf8))
    }

    @Test func concurrent401sProduceExactlyOneRefreshAndAllRetryWithTheRotatedToken() async throws {
        let refreshes = Counter()
        stub401UntilRefresh(refreshes: refreshes, refreshAnswer: Self.rotatedTokensAnswer)
        seedSession()

        // Real concurrency: four profile reads 401 together.
        let names = try await withThrowingTaskGroup(of: String.self) { group in
            for _ in 0 ..< 4 {
                group.addTask { try await api.getProfile().name }
            }
            var collected: [String] = []
            for try await name in group {
                collected.append(name)
            }
            return collected
        }

        #expect(names == ["Ana", "Ana", "Ana", "Ana"])
        // The heart of the test: single flight.
        #expect(refreshes.current == 1)
        // The session now holds the rotated pair, cache and persistence.
        #expect(store.session?.accessToken == "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.example")
        #expect(persisting.load()?.accessToken == "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.example")
        #expect(store.gate == .ready)
    }

    @Test func aRejectedRefreshWipesTheSessionAndTheOriginalCallFails() async throws {
        let refreshes = Counter()
        stub401UntilRefresh(refreshes: refreshes, refreshAnswer: Self.rejectedRefreshAnswer)
        seedSession()

        do {
            _ = try await api.getProfile()
            Issue.record("expected ApiError")
        } catch let error as ApiError {
            #expect(error.code == .unauthorized)
            #expect(error.httpStatus == 401)
        } catch {
            Issue.record("expected ApiError, got \(error)")
        }

        // The rejected refresh must invalidate the session: the reactive
        // gate lands back on onboarding without manual navigation.
        #expect(store.session == nil)
        #expect(store.gate == .needsOnboarding)
        #expect(persisting.load() == nil)
    }

    @Test func anUnreachableRefreshKeepsTheSessionAndGivesUpOnlyTheRequest() async throws {
        let refreshes = Counter()
        // The refresh exchange gets its connection cut: URLError (offline).
        StubURLProtocol.reset { request in
            let path = request.url?.path ?? ""
            if path.hasSuffix("/auth/refresh") {
                refreshes.increment()
                throw URLError(.notConnectedToInternet)
            }
            if path.hasSuffix("/me") {
                return StubResponse(
                    status: 401,
                    headers: ["Content-Type": "application/problem+json"],
                    body: Data(Fixtures.unauthorizedProblem.utf8)
                )
            }
            throw URLError(.unsupportedURL)
        }
        seedSession()

        do {
            _ = try await api.getProfile()
            Issue.record("expected ApiError")
        } catch let error as ApiError {
            // The original request fails with its own 401, but the outage
            // must NOT log the user out.
            #expect(error.code == .unauthorized)
        } catch {
            Issue.record("expected ApiError, got \(error)")
        }

        #expect(store.session?.accessToken == "stale-access-token")
        #expect(store.gate == .ready)
        #expect(persisting.load()?.accessToken == "stale-access-token")
    }

    @Test func theRefreshCallNeverReentersTheRefresher() async throws {
        let refreshes = Counter()
        stub401UntilRefresh(refreshes: refreshes, refreshAnswer: Self.rejectedRefreshAnswer)
        seedSession()

        do {
            _ = try await api.getProfile()
            Issue.record("expected ApiError")
        } catch {
            // The mapped 401 is asserted by the sibling test.
        }

        // One /me 401, one refresh 401, and NOT a second refresh from the
        // recursion: the guard stops the chain.
        #expect(refreshes.current == 1)
        #expect(StubURLProtocol.recordedRequests().count == 2)
        // The refresh is a public endpoint: it never carries the bearer.
        #expect(StubURLProtocol.recordedRequests().last?.headers["Authorization"] == nil)
    }
}
