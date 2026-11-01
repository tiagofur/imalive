import Foundation

/// The auth seam of the onboarding (R-01, issue #76): request the OTP for a
/// phone and verify it into a persisted session. Rotating an expired session
/// is NOT here — it lives in the single-flight TokenRefresher (slice 1), so
/// every call of the app refreshes transparently. Failures surface typed
/// (ApiError with the business code / URLError); the copy per business code
/// is UI's job.
protocol AuthRepository: Sendable {
    func requestOtp(phone: String) async throws

    /// Verifies the code and persists the token pair; the session gate then
    /// lets the user through.
    func verifyOtp(phone: String, code: String) async throws
}

struct NetworkAuthRepository: AuthRepository {
    let api: TranquiAPI
    let sessions: any SessionStoring

    func requestOtp(phone: String) async throws {
        try await api.requestOtp(OtpRequest(phone: phone)) // 204: nothing to carry back
    }

    func verifyOtp(phone: String, code: String) async throws {
        let tokens = try await api.verifyOtp(OtpVerify(phone: phone, code: code))
        await sessions.save(StoredSession(
            accessToken: tokens.accessToken,
            refreshToken: tokens.refreshToken,
            expiresAt: Date().addingTimeInterval(TimeInterval(tokens.expiresInSeconds))
        ))
    }
}
