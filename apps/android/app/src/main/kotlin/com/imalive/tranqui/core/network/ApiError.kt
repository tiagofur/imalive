package com.imalive.tranqui.core.network

/**
 * The only error type that crosses to ViewModels/repos from the API layer
 * (docs/stack/android.md: nothing but mapped errors reach the UI).
 *
 * Built from a `application/problem+json` envelope (RFC 9457). Network
 * failures keep surfacing as plain [java.io.IOException] so callers can
 * distinguish "offline, retry" (IOException) from a business answer
 * ([ApiError]).
 *
 * The [message] is deliberately PII-free (code + status only): safe to log.
 */
class ApiError(
    val code: ApiErrorCode,
    val httpStatus: Int,
    val retryAfterSeconds: Int? = null,
    val type: String? = null,
    val detail: String? = null,
) : Exception("ApiError(${code.name}, http=$httpStatus, retryAfter=$retryAfterSeconds)")

/**
 * Stable business codes of the contract v0.5.0 (Problem.code). The apps
 * switch on these for copy and decisions; anything not in the enum yet maps
 * to [UNKNOWN] so additive contract growth never crashes the client.
 */
enum class ApiErrorCode {
    PHONE_INVALID,
    OTP_INVALID,
    OTP_EXPIRED,
    RATE_LIMITED,
    UNAUTHORIZED,
    VALIDATION_FAILED,
    CONTACT_LIMIT_REACHED,
    CONTACT_NOT_FOUND,
    SOS_PAUSED,
    UNKNOWN,
    ;

    companion object {
        fun from(raw: String?): ApiErrorCode = entries.firstOrNull { it.name == raw } ?: UNKNOWN
    }
}
