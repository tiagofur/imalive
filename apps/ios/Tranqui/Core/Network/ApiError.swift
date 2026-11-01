import Foundation

/// Stable business codes of the contract v0.5.0 (Problem.code). The app
/// switches on these for copy and decisions; anything not in the enum yet
/// maps to `unknown` so additive contract growth never crashes the client.
enum ApiErrorCode: String, Equatable, Sendable {
    case phoneInvalid = "PHONE_INVALID"
    case otpInvalid = "OTP_INVALID"
    case otpExpired = "OTP_EXPIRED"
    case rateLimited = "RATE_LIMITED"
    case unauthorized = "UNAUTHORIZED"
    case validationFailed = "VALIDATION_FAILED"
    case contactLimitReached = "CONTACT_LIMIT_REACHED"
    case contactNotFound = "CONTACT_NOT_FOUND"
    case sosPaused = "SOS_PAUSED"
    case unknown = "UNKNOWN"

    static func from(raw: String?) -> ApiErrorCode {
        guard let raw else {
            return .unknown
        }
        return ApiErrorCode(rawValue: raw) ?? .unknown
    }
}

/// The only error type that crosses to models/repos from the API layer
/// (docs/stack/ios.md: nothing but mapped errors reach the UI). Built from
/// an application/problem+json envelope (RFC 9457). Transport failures keep
/// surfacing as URLError so callers can distinguish "offline, retry" from a
/// business answer. The description is deliberately PII-free (code + status
/// only): safe to log.
struct ApiError: Error, Equatable, Sendable {
    let code: ApiErrorCode
    let httpStatus: Int
    /// Parsed Retry-After (integer seconds); nil when the header is absent
    /// or invalid, so the caller applies its default cooldown (60 s, the
    /// same default as Android).
    let retryAfterSeconds: Int?
    let type: String?
    let detail: String?
}

/// The RFC 9457 problem envelope of the contract (schemas/Problem). Only
/// `code` drives decisions; the rest is carried for diagnostics.
struct Problem: Decodable, Sendable {
    let type: String?
    let title: String?
    let status: Int?
    let detail: String?
    let code: String?
    let instance: String?
}

/// Parses a non-2xx response into an ApiError. Tolerates degraded envelopes:
/// a missing/garbled body or an unknown business `code` must still yield a
/// usable ApiError (unknown + status), never a crash. Retry-After (integer
/// seconds, contract: the 429 rate limit) is carried into the error; an
/// invalid header maps to nil.
enum ProblemEnvelope {
    static func parse(httpStatus: Int, body: Data, retryAfterHeader: String?) -> ApiError {
        let problem = try? JSONDecoder().decode(Problem.self, from: body)
        let seconds = retryAfterHeader.map { $0.trimmingCharacters(in: .whitespaces) }.flatMap(Int.init)
        return ApiError(
            code: ApiErrorCode.from(raw: problem?.code),
            httpStatus: httpStatus,
            retryAfterSeconds: seconds,
            type: problem?.type,
            detail: problem?.detail
        )
    }
}
