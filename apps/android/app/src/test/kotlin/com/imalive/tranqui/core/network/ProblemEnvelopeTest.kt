package com.imalive.tranqui.core.network

import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Test

// Problem-envelope parsing with the exact `examples` of openapi.yaml v0.5.0
// as fixtures (ADR-0003: the contract is the fixture source).
class ProblemEnvelopeTest {
    private val json = ApiClientFactory.json()

    @Test
    fun `429 rate-limited carries the business code and the Retry-After`() {
        // contracts/openapi.yaml, responses.TooManyRequests example.
        val body =
            """
            {
              "type": "https://api.tranqui.app/problems/rate-limited",
              "title": "Too Many Requests",
              "status": 429,
              "detail": "Too many requests; slow down and retry later.",
              "code": "RATE_LIMITED",
              "instance": "/v1/auth/otp/request"
            }
            """.trimIndent()

        val error = ProblemEnvelope.parse(json, httpStatus = 429, body = body, retryAfterHeader = "60")

        assertEquals(ApiErrorCode.RATE_LIMITED, error.code)
        assertEquals(429, error.httpStatus)
        assertEquals(60, error.retryAfterSeconds)
        assertEquals("Too many requests; slow down and retry later.", error.detail)
    }

    @Test
    fun `otp-invalid problem maps the code without retry-after`() {
        // contracts/openapi.yaml, POST /auth/otp/verify 400 "invalid" example.
        val body =
            """
            {
              "type": "https://api.tranqui.app/problems/otp-invalid",
              "title": "Invalid code",
              "status": 400,
              "detail": "The code does not match or has too many attempts.",
              "code": "OTP_INVALID",
              "instance": "/v1/auth/otp/verify"
            }
            """.trimIndent()

        val error = ProblemEnvelope.parse(json, httpStatus = 400, body = body, retryAfterHeader = null)

        assertEquals(ApiErrorCode.OTP_INVALID, error.code)
        assertEquals(400, error.httpStatus)
        assertNull(error.retryAfterSeconds)
        assertEquals("The code does not match or has too many attempts.", error.detail)
    }

    @Test
    fun `a business code the client does not know yet maps to UNKNOWN`() {
        val body = """{"type":"https://api.tranqui.app/problems/future","title":"Future","status":422,"code":"SOME_FUTURE_CODE"}"""

        val error = ProblemEnvelope.parse(json, httpStatus = 422, body = body)

        assertEquals(ApiErrorCode.UNKNOWN, error.code)
        assertEquals(422, error.httpStatus)
    }

    @Test
    fun `a non-problem body still yields a usable error`() {
        val error = ProblemEnvelope.parse(json, httpStatus = 502, body = "<html>bad gateway</html>")

        assertEquals(ApiErrorCode.UNKNOWN, error.code)
        assertEquals(502, error.httpStatus)
    }

    @Test
    fun `an invalid Retry-After header maps to null`() {
        val error =
            ProblemEnvelope.parse(
                json,
                httpStatus = 429,
                body = """{"type":"about:blank","title":"t","status":429,"code":"RATE_LIMITED"}""",
                retryAfterHeader = " in 2 minutes ",
            )

        assertNull(error.retryAfterSeconds)
        assertEquals(ApiErrorCode.RATE_LIMITED, error.code)
    }

    @Test
    fun `the error message stays PII-free`() {
        // Privacy: the message is what lands in crash reports and logs.
        val body =
            """
            {"type":"https://api.tranqui.app/problems/phone-invalid","title":"Invalid phone number","status":400,
            "detail":"Phone must be E.164, e.g. +5215512345678.","code":"PHONE_INVALID","instance":"/v1/auth/otp/request"}
            
            """.trimIndent()

        val error = ProblemEnvelope.parse(json, httpStatus = 400, body = body)

        assertEquals(ApiErrorCode.PHONE_INVALID, error.code)
        assertEquals("ApiError(PHONE_INVALID, http=400, retryAfter=null)", error.message)
    }
}
