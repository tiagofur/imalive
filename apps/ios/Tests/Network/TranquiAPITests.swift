import Foundation
import Testing
@testable import Tranqui

/// The contract's own examples (contracts/openapi.yaml v0.5.0), verbatim:
/// they are the fixtures AND the drift guard of the hand-written client
/// (ADR-0003). If the contract example changes shape, these tests break.
enum Fixtures {
    static let tokens = """
    {
      "accessToken": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.example",
      "refreshToken": "7f3a9c2e5b1d4a6f8e0c3b5d7a9e1f2a4",
      "expiresInSeconds": 900
    }
    """

    static let profile = """
    {
      "id": "01934f2e-7a1b-7c3d-9e4f-1a2b3c4d5e6f",
      "name": "Ana",
      "phone": "+5215512345678",
      "checkinTime": "21:00",
      "timezone": "America/Mexico_City",
      "escalationWindowMinutes": 60,
      "streakDays": 12
    }
    """

    /// contracts/openapi.yaml: the UserProfile schema example with the active
    /// pause (pauseUntil, R-11).
    static let pausedProfile = """
    {
      "id": "01934f2e-7a1b-7c3d-9e4f-1a2b3c4d5e6f",
      "name": "Ana",
      "phone": "+5215512345678",
      "checkinTime": "21:00",
      "timezone": "America/Mexico_City",
      "escalationWindowMinutes": 60,
      "streakDays": 12,
      "pauseUntil": "2026-11-15T06:00:00Z"
    }
    """

    /// The same profile with a garbage timestamp where pauseUntil goes.
    static let degradedPauseProfile = """
    {
      "id": "01934f2e-7a1b-7c3d-9e4f-1a2b3c4d5e6f",
      "name": "Ana",
      "phone": "+5215512345678",
      "checkinTime": "21:00",
      "timezone": "America/Mexico_City",
      "escalationWindowMinutes": 60,
      "streakDays": 12,
      "pauseUntil": "the day after the pause started"
    }
    """

    static let rateLimitedProblem = """
    {
      "type": "https://api.tranqui.app/problems/rate-limited",
      "title": "Too Many Requests",
      "status": 429,
      "detail": "Too many requests; slow down and retry later.",
      "code": "RATE_LIMITED",
      "instance": "/v1/auth/otp/request"
    }
    """

    static let otpExpiredProblem = """
    {
      "type": "https://api.tranqui.app/problems/otp-expired",
      "title": "Code expired",
      "status": 400,
      "detail": "The code expired; request a new one.",
      "code": "OTP_EXPIRED",
      "instance": "/v1/auth/otp/verify"
    }
    """

    static let unauthorizedProblem = """
    {
      "type": "https://api.tranqui.app/problems/unauthorized",
      "title": "Unauthorized",
      "status": 401,
      "detail": "Missing, expired or invalid credentials.",
      "code": "UNAUTHORIZED",
      "instance": "/v1/me"
    }
    """

    static let checkin = """
    {
      "date": "2026-10-28",
      "checkedInAt": "2026-10-28T07:58:00Z",
      "replayed": false
    }
    """

    /// contracts/openapi.yaml, GET /contacts 200 example.
    static let contactsList = """
    {
      "contacts": [
        {
          "id": "01934f2f-8a2c-7d4e-af5a-2b3c4d5e6f7a",
          "name": "María",
          "phone": "+5215598765432",
          "createdAt": "2026-10-20T15:30:00Z"
        },
        {
          "id": "01934f30-9b3d-7e5f-b06b-3c4d5e6f7a8b",
          "name": "Jorge",
          "phone": "+5215511223344",
          "createdAt": "2026-10-21T10:00:00Z"
        }
      ]
    }
    """

    /// contracts/openapi.yaml, POST /contacts 201 example.
    static let contact = """
    {
      "id": "01934f2f-8a2c-7d4e-af5a-2b3c4d5e6f7a",
      "name": "María",
      "phone": "+5215598765432",
      "createdAt": "2026-10-20T15:30:00Z"
    }
    """

    /// contracts/openapi.yaml, POST /contacts 422 example (verbatim, the
    /// Spanish detail included).
    static let contactLimitProblem = """
    {
      "type": "https://api.tranqui.app/problems/contact-limit-reached",
      "title": "Límite de contactos alcanzado",
      "status": 422,
      "detail": "El plan gratuito permite hasta 1 contacto. Actualiza a Premium.",
      "code": "CONTACT_LIMIT_REACHED",
      "instance": "/v1/contacts"
    }
    """
}

/// Slice 1 of #76: the real client against StubURLProtocol (the idiomatic
/// MockWebServer replacement, zero dependencies). Request shapes, bearer
/// attach and the RFC 9457 error mapping live here.
@Suite("TranquiAPI")
struct TranquiAPITests {
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

    @Test func otpRequestPostsTheE164PhoneAndAcceptsThe204() async throws {
        stub(204, "")

        try await makeAPI().requestOtp(OtpRequest(phone: "+5215512345678"))

        let request = StubURLProtocol.lastRequest
        #expect(request?.method == "POST")
        #expect(request?.path == "/v1/auth/otp/request")
        #expect(request?.body == Data("{\"phone\":\"+5215512345678\"}".utf8))
        #expect(request?.headers["Content-Type"] == "application/json")
        // Auth endpoints are public: no Authorization header even with a session.
        #expect(request?.headers["Authorization"] == nil)
    }

    @Test func otpRequestMaps429RetryAfterIntoTheApiError() async throws {
        stub(429, Fixtures.rateLimitedProblem, headers: ["Retry-After": "60"])

        do {
            _ = try await makeAPI().requestOtp(OtpRequest(phone: "+5215512345678"))
            Issue.record("expected ApiError")
        } catch let error as ApiError {
            #expect(error.code == .rateLimited)
            #expect(error.httpStatus == 429)
            #expect(error.retryAfterSeconds == 60)
        } catch {
            Issue.record("expected ApiError, got \(error)")
        }
    }

    @Test func otpRequestWithoutRetryAfterLeavesTheCooldownNil() async throws {
        stub(429, Fixtures.rateLimitedProblem)

        do {
            _ = try await makeAPI().requestOtp(OtpRequest(phone: "+5215512345678"))
            Issue.record("expected ApiError")
        } catch let error as ApiError {
            // The caller applies the 60 s default cooldown, not the client.
            #expect(error.retryAfterSeconds == nil)
        } catch {
            Issue.record("expected ApiError, got \(error)")
        }
    }

    @Test func otpVerifyPostsPhoneAndCodeAndParsesTheTokenPair() async throws {
        stub(200, Fixtures.tokens)

        let tokens = try await makeAPI().verifyOtp(OtpVerify(phone: "+5215512345678", code: "482913"))

        #expect(tokens.accessToken == "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.example")
        #expect(tokens.refreshToken == "7f3a9c2e5b1d4a6f8e0c3b5d7a9e1f2a4")
        #expect(tokens.expiresInSeconds == 900)
        let request = StubURLProtocol.lastRequest
        #expect(request?.path == "/v1/auth/otp/verify")
        // The client canonicalizes with sorted keys, so the bytes are stable.
        #expect(request?.body == Data("{\"code\":\"482913\",\"phone\":\"+5215512345678\"}".utf8))
    }

    @Test func otpVerifyMapsTheOtpExpiredProblemToApiError() async throws {
        stub(400, Fixtures.otpExpiredProblem)

        do {
            _ = try await makeAPI().verifyOtp(OtpVerify(phone: "+5215512345678", code: "111111"))
            Issue.record("expected ApiError")
        } catch let error as ApiError {
            #expect(error.code == .otpExpired)
            #expect(error.httpStatus == 400)
            #expect(error.retryAfterSeconds == nil)
        } catch {
            Issue.record("expected ApiError, got \(error)")
        }
    }

    @Test func protectedCallsCarryTheBearerTokenOfTheSession() async throws {
        let session = StoredSession(
            accessToken: "access-token-1",
            refreshToken: "refresh-token-1",
            expiresAt: Date(timeIntervalSince1970: 0)
        )
        sessions.store(session)
        stub(200, Fixtures.profile)

        let profile = try await makeAPI().getProfile()

        #expect(profile.name == "Ana")
        #expect(profile.streakDays == 12)
        #expect(profile.checkinTime == "21:00")
        #expect(StubURLProtocol.lastRequest?.headers["Authorization"] == "Bearer access-token-1")
    }

    @Test func getProfileDecodesTheActivePauseOfTheSchemaExample() async throws {
        stub(200, Fixtures.pausedProfile)

        let profile = try await makeAPI().getProfile()

        let formatter = ISO8601DateFormatter()
        #expect(profile.pauseUntil == formatter.date(from: "2026-11-15T06:00:00Z"))
    }

    @Test func getProfileReadsADegradedPauseUntilAsNoPause() async throws {
        stub(200, Fixtures.degradedPauseProfile)

        let profile = try await makeAPI().getProfile()

        #expect(profile.streakDays == 12)
        #expect(profile.pauseUntil == nil)
    }

    @Test func withoutASessionProtectedCallsGoOutAnonymousAnd401MapsToApiError() async throws {
        stub(401, Fixtures.unauthorizedProblem)

        do {
            _ = try await makeAPI().getProfile()
            Issue.record("expected ApiError")
        } catch let error as ApiError {
            #expect(error.code == .unauthorized)
            #expect(error.httpStatus == 401)
            #expect(StubURLProtocol.lastRequest?.headers["Authorization"] == nil)
        } catch {
            Issue.record("expected ApiError, got \(error)")
        }
    }

    @Test func anOfflineCallSurfacesAsURLErrorNeverAsApiError() async throws {
        StubURLProtocol.reset { _ in
            throw URLError(.notConnectedToInternet)
        }

        do {
            _ = try await makeAPI().requestOtp(OtpRequest(phone: "+5215512345678"))
            Issue.record("expected a transport error")
        } catch {
            #expect(error is URLError)
            #expect(!(error is ApiError))
        }
    }

    @Test func patchProfileSendsTheExactBodyAndDecodesTheFullProfile() async throws {
        let updated = Fixtures.profile.replacingOccurrences(
            of: "\"checkinTime\": \"21:00\"",
            with: "\"checkinTime\": \"22:00\""
        )
        stub(200, updated)

        let patch = ProfilePatch(checkinTime: "22:00", escalationWindowMinutes: 90)
        let profile = try await makeAPI().updateProfile(patch)

        // Only what the user touched goes on the wire (nils are skipped).
        let expectedBody = Data("{\"checkinTime\":\"22:00\",\"escalationWindowMinutes\":90}".utf8)
        #expect(StubURLProtocol.lastRequest?.method == "PATCH")
        #expect(StubURLProtocol.lastRequest?.body == expectedBody)
        #expect(profile.checkinTime == "22:00")
        #expect(profile.streakDays == 12)
    }

    @Test func createCheckinSendsTheIdempotencyKeyAndParsesTheSignal() async throws {
        stub(201, Fixtures.checkin)

        let checkin = try await makeAPI().createCheckin(idempotencyKey: "01934f31-0a4e-7f60-c17c-4d5e6f7a8b9c")

        #expect(checkin.date == "2026-10-28")
        #expect(checkin.replayed == false)
        let request = StubURLProtocol.lastRequest
        #expect(request?.method == "POST")
        #expect(request?.path == "/v1/checkins")
        #expect(request?.headers["Idempotency-Key"] == "01934f31-0a4e-7f60-c17c-4d5e6f7a8b9c")
    }
}
