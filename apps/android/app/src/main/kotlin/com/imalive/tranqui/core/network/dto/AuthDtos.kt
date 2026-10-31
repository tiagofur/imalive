package com.imalive.tranqui.core.network.dto

import kotlinx.serialization.Serializable

// DTOs named 1:1 with the contract schemas (ADR-0003, openapi.yaml v0.5.0).
// Fixtures for tests come from the spec's own examples. No PII beyond the
// phone the user typed, and none of it is ever logged (AuthInterceptor).

// Request an OTP code for the phone (POST /auth/otp/request, 204).
@Serializable
data class OtpRequest(
    val phone: String,
)

// Verify the code received by WhatsApp/SMS (POST /auth/otp/verify).
@Serializable
data class OtpVerify(
    val phone: String,
    val code: String,
)

// Session token pair issued by verify and by refresh. No PII: the profile
// comes from GET /me.
@Serializable
data class Tokens(
    val accessToken: String,
    val refreshToken: String,
    val expiresInSeconds: Int,
)

// Rotate the session with a valid refresh token (POST /auth/refresh).
@Serializable
data class RefreshRequest(
    val refreshToken: String,
)
