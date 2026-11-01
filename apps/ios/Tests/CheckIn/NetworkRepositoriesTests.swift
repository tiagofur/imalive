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

    // MARK: - auth (POST /auth/otp/*)

    @Test func otpRequestPostsThePhoneAndAcceptsThe204Anonymously() async throws {
        stub(204, "")

        try await NetworkAuthRepository(api: makeAPI(), sessions: sessions).requestOtp(phone: "+5215512345678")

        let request = StubURLProtocol.lastRequest
        #expect(request?.path == "/v1/auth/otp/request")
        #expect(request?.body == Data("{\"phone\":\"+5215512345678\"}".utf8))
        #expect(request?.headers["Authorization"] == nil)
    }

    @Test func otpVerifyPersistsTheTokenPairInTheSession() async throws {
        stub(200, Fixtures.tokens)
        let before = Date()

        try await NetworkAuthRepository(api: makeAPI(), sessions: sessions)
            .verifyOtp(phone: "+5215512345678", code: "482913")

        let saved = try #require(sessions.currentSession)
        #expect(saved.accessToken == "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.example")
        #expect(saved.refreshToken == "7f3a9c2e5b1d4a6f8e0c3b5d7a9e1f2a4")
        // now + expiresInSeconds, within the second the test took.
        let after = Date()
        #expect(saved.expiresAt > before.addingTimeInterval(900))
        #expect(saved.expiresAt < after.addingTimeInterval(900))
    }

    @Test func otpVerifyWithAnExpiredCodeSurfacesOTPExpiredAndKeepsTheStoredSession() async throws {
        sessions.store(makeSession())
        stub(400, Fixtures.otpExpiredProblem)

        do {
            try await NetworkAuthRepository(api: makeAPI(), sessions: sessions)
                .verifyOtp(phone: "+5215512345678", code: "111111")
            Issue.record("expected ApiError")
        } catch let error as ApiError {
            #expect(error.code == .otpExpired)
            #expect(error.httpStatus == 400)
            // The failed verify does not touch the session that was there.
            #expect(sessions.currentSession?.accessToken == "access-token-1")
        } catch {
            Issue.record("expected ApiError, got \(error)")
        }
    }

    // MARK: - contacts (GET/POST /v1/contacts)

    @Test func contactsListsTheCircleFromTheContractExample() async throws {
        sessions.store(makeSession())
        stub(200, Fixtures.contactsList)

        let circle = try await NetworkContactsRepository(api: makeAPI()).contacts()

        #expect(circle.count == 2)
        #expect(circle[0].name == "María")
        #expect(circle[0].phone == "+5215598765432")
        #expect(circle[1].name == "Jorge")
        #expect(StubURLProtocol.lastRequest?.path == "/v1/contacts")
    }

    @Test func createPostsNameAndPhoneAndReturnsThe201Contact() async throws {
        sessions.store(makeSession())
        stub(201, Fixtures.contact)

        let contact = try await NetworkContactsRepository(api: makeAPI())
            .create(name: "María", phone: "+5215598765432")

        #expect(contact.id == "01934f2f-8a2c-7d4e-af5a-2b3c4d5e6f7a")
        #expect(contact.name == "María")
        #expect(contact.phone == "+5215598765432")
        #expect(
            StubURLProtocol.lastRequest?.body == Data("{\"name\":\"María\",\"phone\":\"+5215598765432\"}".utf8)
        )
    }

    @Test func createMapsContactLimitReachedToATypedApiError() async throws {
        sessions.store(makeSession())
        stub(422, Fixtures.contactLimitProblem)

        do {
            _ = try await NetworkContactsRepository(api: makeAPI())
                .create(name: "Jorge", phone: "+5215511223344")
            Issue.record("expected ApiError")
        } catch let error as ApiError {
            #expect(error.code == .contactLimitReached)
            #expect(error.httpStatus == 422)
        } catch {
            Issue.record("expected ApiError, got \(error)")
        }
    }
}
