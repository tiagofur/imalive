package com.imalive.tranqui.core.network

import com.imalive.tranqui.core.network.dto.Problem
import kotlinx.serialization.json.Json

/**
 * Parses a non-2xx response (RFC 9457 problem envelope) into an [ApiError].
 *
 * Tolerates degraded envelopes: a missing/garbled body or an unknown
 * business `code` must still yield a usable [ApiError] (UNKNOWN + status),
 * never a crash. `Retry-After` (integer seconds, contract: 429 rate limit)
 * is carried into the error; an invalid header maps to null and the caller
 * applies its default cooldown.
 */
object ProblemEnvelope {
    fun parse(
        json: Json,
        httpStatus: Int,
        body: String?,
        retryAfterHeader: String? = null,
    ): ApiError {
        val problem = body?.let { raw -> runCatching { json.decodeFromString<Problem>(raw) }.getOrNull() }
        return ApiError(
            code = ApiErrorCode.from(problem?.code),
            httpStatus = httpStatus,
            retryAfterSeconds = retryAfterHeader?.trim()?.toIntOrNull(),
            type = problem?.type,
            detail = problem?.detail,
        )
    }
}
