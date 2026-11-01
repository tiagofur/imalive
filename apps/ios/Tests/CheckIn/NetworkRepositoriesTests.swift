import Foundation
import Testing
@testable import Tranqui

/// Slice 2 of #76: the real repositories against StubURLProtocol (the same
/// pattern as TranquiAPITests — the idiomatic MockWebServer replacement),
/// with the contract's own examples as fixtures. The repos surface typed
/// failures (ApiError with the business code / URLError); turning them into
/// UI states is the models' job.
@Suite("NetworkRepositories")
struct NetworkRepositoriesTests {
    private let sessions = FakeSessionStore()

    private func makeAPI() -> TranquiAPI {
        let configuration = URLSessionConfiguration.ephemeral
        configuration.protocolClasses = [StubURLProtocol.self]
        return TranquiAPI(
            baseURL: URL(string: "https://stub.local/v1")!,
            sessions: sessions,
            transport: URLSession(configuration: configuration)
        )
    }

    private func stub(_ status: Int, _ body: String, headers: [String: String] = [:]) {
        StubURLProtocol.reset { _ in
            StubResponse(status: status, headers: headers, body: Data(body.utf8))
        }
    }

    private func makeSession() -> StoredSession {
        StoredSession(
            accessToken: "access-token-1",
            refreshToken: "refresh-token-1",
            expiresAt: Date(timeIntervalSince1970: 0)
        )
    }

    // MARK: - profile (GET/PATCH /v1/me)

    @Test func profileReadsGETMeWithTheBearerAndParsesTheContractExample() async throws {
        sessions.store(makeSession())
        stub(200, Fixtures.profile)

        let profile = try await NetworkProfileRepository(api: makeAPI()).profile()

        #expect(profile.streakDays == 12)
        #expect(profile.checkinTime == "21:00")
        #expect(profile.pauseUntil == nil)
        let request = StubURLProtocol.lastRequest
        #expect(request?.path == "/v1/me")
        #expect(request?.headers["Authorization"] == "Bearer access-token-1")
    }

    @Test func profileMapsThePauseWindowOfTheSchemaExampleIntoADate() async throws {
        sessions.store(makeSession())
        stub(200, Fixtures.pausedProfile)

        let profile = try await NetworkProfileRepository(api: makeAPI()).profile()

        let formatter = ISO8601DateFormatter()
        let expected = try #require(formatter.date(from: "2026-11-15T06:00:00Z"))
        #expect(profile.pauseUntil == expected)
        #expect(profile.streakDays == 12)
    }

    @Test func profileSurvivesADegradedPauseUntilAsNoPause() async throws {
        sessions.store(makeSession())
        stub(200, Fixtures.degradedPauseProfile)

        let profile = try await NetworkProfileRepository(api: makeAPI()).profile()

        #expect(profile.streakDays == 12)
        #expect(profile.pauseUntil == nil)
    }

    @Test func profileSurfacesThe401ProblemAsATypedApiErrorNeverSwallowed() async {
        sessions.store(makeSession())
        stub(401, Fixtures.unauthorizedProblem)

        do {
            _ = try await NetworkProfileRepository(api: makeAPI()).profile()
            Issue.record("expected ApiError")
        } catch let error as ApiError {
            #expect(error.code == .unauthorized)
            #expect(error.httpStatus == 401)
        } catch {
            Issue.record("expected ApiError, got \(error)")
        }
    }

    @Test func updatePostsTheRitualPatchWithOnlyTheFieldsTheOnboardingSets() async throws {
        sessions.store(makeSession())
        stub(200, Fixtures.profile)

        try await NetworkProfileRepository(api: makeAPI())
            .update(ProfilePatch(name: "Ana", checkinTime: "21:00", timezone: "America/Mexico_City"))

        let request = StubURLProtocol.lastRequest
        #expect(request?.method == "PATCH")
        #expect(request?.path == "/v1/me")
        // Only what the user touched goes on the wire (nils are skipped);
        // the encoder canonicalizes with sorted keys, so the bytes are exact.
        let expectedBody = Data(
            "{\"checkinTime\":\"21:00\",\"name\":\"Ana\",\"timezone\":\"America/Mexico_City\"}".utf8
        )
        #expect(request?.body == expectedBody)
    }
}
