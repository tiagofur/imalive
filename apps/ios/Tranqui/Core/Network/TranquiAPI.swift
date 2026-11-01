import Foundation

/// The thin hand-written client for the week-1 surface of the contract
/// (openapi.yaml v0.5.0, ADR-0003) — the iOS mirror of Android's TranquiApi.
/// Paths resolve against the base URL (…/v1). The openapi-generator client
/// returns when the surface outgrows this file (docs/stack/ios.md documents
/// the threshold); a generic Endpoint abstraction does not pay for 8
/// operations and costs the call sites their greppability.
///
/// Every failure maps to the two errors the app understands: URLError
/// (offline/transport) and ApiError (an RFC 9457 problem answer, with the
/// business code and the parsed Retry-After). Auth endpoints (`security: []`
/// in the contract) never carry the bearer header; protected calls attach it
/// from the session cache and, on a 401, retry once through the single-flight
/// refresh (TokenRefresher).
actor TranquiAPI {
    private let baseURL: URL
    private let transport: URLSession
    private let sessions: any SessionStoring
    private let refresher: TokenRefresher

    init(
        baseURL: URL = URL(string: "https://api.tranqui.app/v1")!,
        sessions: any SessionStoring,
        transport: URLSession = URLSession(configuration: .tranqui)
    ) {
        self.baseURL = baseURL
        self.sessions = sessions
        self.transport = transport
        refresher = TokenRefresher(sessions: sessions)
    }

    // MARK: - Week-1 operations (contract v0.5.0)

    /// POST /auth/otp/request → 204. Always 204 per contract, even for
    /// unknown phones (no account enumeration).
    func requestOtp(_ body: OtpRequest) async throws {
        let request = try request("POST", "auth/otp/request", body: body)
        _ = try await send(request, refreshTokenOn401: false)
    }

    /// POST /auth/otp/verify → Tokens (the session pair to persist).
    func verifyOtp(_ body: OtpVerify) async throws -> Tokens {
        let request = try request("POST", "auth/otp/verify", body: body)
        let (data, _) = try await send(request, refreshTokenOn401: false)
        return try Self.decode(Tokens.self, from: data)
    }

    /// POST /auth/refresh → rotated Tokens. The refresher drives this; the
    /// refresh itself never refreshes again (recursion guard) and carries no
    /// bearer header (public endpoint).
    func refreshSession(_ body: RefreshRequest) async throws -> Tokens {
        let request = try request("POST", "auth/refresh", body: body)
        let (data, _) = try await send(request, refreshTokenOn401: false)
        return try Self.decode(Tokens.self, from: data)
    }

    /// GET /me → the profile (name, phone, check-in prefs, streak).
    func getProfile() async throws -> UserProfile {
        let (data, _) = try await send(request("GET", "me"), refreshTokenOn401: true)
        return try Self.decode(UserProfile.self, from: data)
    }

    /// PATCH /me → the updated profile; omitted fields keep their values
    /// (the #88 fix guarantees streakDays on every 200).
    func updateProfile(_ body: ProfilePatch) async throws -> UserProfile {
        let request = try request("PATCH", "me", body: body)
        let (data, _) = try await send(request, refreshTokenOn401: true)
        return try Self.decode(UserProfile.self, from: data)
    }

    /// GET /contacts → the trusted circle (1-5, display order).
    func listContacts() async throws -> ContactList {
        let (data, _) = try await send(request("GET", "contacts"), refreshTokenOn401: true)
        return try Self.decode(ContactList.self, from: data)
    }

    /// POST /contacts → the added contact (the backend sends the welcome).
    func createContact(_ body: ContactCreate) async throws -> Contact {
        let request = try request("POST", "contacts", body: body)
        let (data, _) = try await send(request, refreshTokenOn401: true)
        return try Self.decode(Contact.self, from: data)
    }

    /// POST /checkins → today's "I'm OK" signal, idempotent via the
    /// client-generated Idempotency-Key (201 first signal, 200 replay).
    func createCheckin(idempotencyKey: String? = nil) async throws -> Checkin {
        let (data, _) = try await send(
            request("POST", "checkins", idempotencyKey: idempotencyKey),
            refreshTokenOn401: true
        )
        return try Self.decode(Checkin.self, from: data)
    }

    // MARK: - Pipeline

    /// One request lifecycle: authorize → send → map. A 401 on a protected
    /// call with `refreshTokenOn401` rotates once through the single-flight
    /// refresher and retries with the fresh pair; a second 401 is terminal
    /// (a fresh token rejected means the session is dead server-side). A
    /// rejected refresh surfaces the original 401; an unreachable refresh
    /// does too — the session survives for the next attempt either way.
    private func send(_ request: URLRequest, refreshTokenOn401: Bool) async throws -> (Data, HTTPURLResponse) {
        let prepared = await authorized(request)
        let (data, response) = try await transport.data(for: prepared)
        guard let http = response as? HTTPURLResponse else {
            throw URLError(.badServerResponse)
        }
        if http.statusCode == 401, refreshTokenOn401, let stale = Self.bearerToken(of: prepared) {
            let outcome = await refresher.refreshedSession(staleAccessToken: stale) { refreshToken in
                try await self.refreshTokens(refreshToken)
            }
            switch outcome {
            case let .rotated(fresh):
                var retried = prepared
                retried.setValue("Bearer \(fresh.accessToken)", forHTTPHeaderField: "Authorization")
                // One retry, never two: the guard keeps a second 401 terminal.
                return try await send(retried, refreshTokenOn401: false)
            case .rejected, .unavailable:
                break // the original 401 is the answer
            }
        }
        guard (200 ..< 300).contains(http.statusCode) else {
            throw ProblemEnvelope.parse(
                httpStatus: http.statusCode,
                body: data,
                retryAfterHeader: http.value(forHTTPHeaderField: "Retry-After")
            )
        }
        return (data, http)
    }

    /// The raw refresh exchange the refresher drives: no bearer (public
    /// endpoint) and no nested refresh (a 401 here means the session is
    /// dead — an ApiError, and the refresher wipes the session on it).
    private func refreshTokens(_ refreshToken: String) async throws -> Tokens {
        let (data, _) = try await send(
            request("POST", "auth/refresh", body: RefreshRequest(refreshToken: refreshToken)),
            refreshTokenOn401: false
        )
        return try Self.decode(Tokens.self, from: data)
    }

    /// The interceptor-equivalent: attaches `Authorization: Bearer …` to
    /// every protected call from the session cache (in-memory, never a
    /// per-request Keychain hit). Auth endpoints go out untouched even with
    /// a session. The client logs nothing, so the token never reaches a log.
    private func authorized(_ request: URLRequest) async -> URLRequest {
        var request = request
        guard !Self.isAuthPath(request.url) else {
            return request
        }
        if let token = await sessions.cachedSession()?.accessToken {
            request.setValue("Bearer \(token)", forHTTPHeaderField: "Authorization")
        }
        return request
    }

    private func request(_ method: String, _ path: String, idempotencyKey: String? = nil) -> URLRequest {
        var request = URLRequest(url: baseURL.appending(path: path))
        request.httpMethod = method
        if let idempotencyKey {
            request.setValue(idempotencyKey, forHTTPHeaderField: "Idempotency-Key")
        }
        return request
    }

    private func request(_ method: String, _ path: String, body: some Encodable) throws -> URLRequest {
        var request = request(method, path)
        request.httpBody = try Self.encode(body)
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        return request
    }

    /// The wire encoder. Sorted keys keep the output canonical: key order is
    /// irrelevant to the contract, but a deterministic body makes the exact
    /// bytes assertable in the tests (JSONEncoder's default order is
    /// hash-based and changes between runs).
    static func encode(_ value: some Encodable) throws -> Data {
        let encoder = JSONEncoder()
        encoder.outputFormatting = [.sortedKeys]
        return try encoder.encode(value)
    }

    private static func decode<T: Decodable>(_ type: T.Type, from data: Data) throws -> T {
        try JSONDecoder.wire.decode(type, from: data)
    }

    static func isAuthPath(_ url: URL?) -> Bool {
        url?.path.contains("/auth/") == true
    }

    static func bearerToken(of request: URLRequest) -> String? {
        guard let header = request.value(forHTTPHeaderField: "Authorization"), header.hasPrefix("Bearer ") else {
            return nil
        }
        return String(header.dropFirst("Bearer ".count))
    }
}

extension URLSessionConfiguration {
    /// Production transport: no cookies (the JWT rides the Authorization
    /// header, never the cookie jar) and fail-fast timeouts so an offline
    /// state reads as URLError instead of a hang.
    static var tranqui: URLSessionConfiguration {
        let configuration = URLSessionConfiguration.ephemeral
        configuration.httpShouldSetCookies = false
        configuration.httpCookieAcceptPolicy = .never
        configuration.timeoutIntervalForRequest = 15
        configuration.timeoutIntervalForResource = 60
        return configuration
    }
}

extension JSONDecoder {
    /// The contract's RFC 3339 timestamps, with or without fractional
    /// seconds (Go's RFC 3339Nano emits fractionals only when nonzero).
    /// Formatters are built per decode: DateFormatter is not Sendable and
    /// Swift 6 strict concurrency forbids sharing one.
    static var wire: JSONDecoder {
        let decoder = JSONDecoder()
        decoder.dateDecodingStrategy = .custom { decoder in
            let raw = try decoder.singleValueContainer().decode(String.self)
            let fractional = ISO8601DateFormatter()
            fractional.formatOptions = [.withInternetDateTime, .withFractionalSeconds]
            if let date = fractional.date(from: raw) {
                return date
            }
            if let date = ISO8601DateFormatter().date(from: raw) {
                return date
            }
            let path = decoder.codingPath.map(\.stringValue).joined(separator: ".")
            throw DecodingError.dataCorrupted(.init(
                codingPath: decoder.codingPath,
                debugDescription: "Expected an RFC 3339 date-time at \(path), got \(raw)"
            ))
        }
        return decoder
    }
}
