package com.imalive.tranqui.core.network

import com.imalive.tranqui.core.network.dto.Checkin
import com.imalive.tranqui.core.network.dto.Contact
import com.imalive.tranqui.core.network.dto.ContactCreate
import com.imalive.tranqui.core.network.dto.ContactList
import com.imalive.tranqui.core.network.dto.OtpRequest
import com.imalive.tranqui.core.network.dto.OtpVerify
import com.imalive.tranqui.core.network.dto.ProfilePatch
import com.imalive.tranqui.core.network.dto.RefreshRequest
import com.imalive.tranqui.core.network.dto.Tokens
import com.imalive.tranqui.core.network.dto.UserProfile
import kotlinx.serialization.json.Json
import retrofit2.Response
import retrofit2.http.Body
import retrofit2.http.GET
import retrofit2.http.Header
import retrofit2.http.PATCH
import retrofit2.http.POST
import java.io.IOException

/**
 * The thin hand-written client for the week-1 surface of the contract
 * (openapi.yaml v0.5.0, ADR-0003). Relative paths resolve against the base
 * URL (https://…/v1/). The generated openapi-generator client returns when
 * the surface outgrows this file (docs/stack/android.md documents the
 * threshold).
 */
interface TranquiApi {
    // auth (public endpoints, security: [] in the contract).
    @POST("auth/otp/request")
    suspend fun requestOtp(
        @Body body: OtpRequest,
    ): Response<Unit>

    @POST("auth/otp/verify")
    suspend fun verifyOtp(
        @Body body: OtpVerify,
    ): Response<Tokens>

    @POST("auth/refresh")
    suspend fun refreshSession(
        @Body body: RefreshRequest,
    ): Response<Tokens>

    // profile.
    @GET("me")
    suspend fun getProfile(): Response<UserProfile>

    @PATCH("me")
    suspend fun updateProfile(
        @Body body: ProfilePatch,
    ): Response<UserProfile>

    // contacts.
    @GET("contacts")
    suspend fun listContacts(): Response<ContactList>

    @POST("contacts")
    suspend fun createContact(
        @Body body: ContactCreate,
    ): Response<Contact>

    // checkins.
    @POST("checkins")
    suspend fun createCheckin(
        @Header("Idempotency-Key") idempotencyKey: String?,
    ): Response<Checkin>
}

/**
 * Runs one API call and maps every failure to the two error types the app
 * understands: [IOException] (offline/transport) and [ApiError] (an RFC 9457
 * problem answer, with the business code and Retry-After). Success returns
 * the decoded body; [Unit] endpoints (204 No Content) return Unit.
 */
@Suppress("UNCHECKED_CAST")
suspend inline fun <reified T> apiCall(
    json: Json,
    call: suspend () -> Response<T>,
): T {
    val response = call()
    if (!response.isSuccessful) {
        throw ProblemEnvelope.parse(
            json = json,
            httpStatus = response.code(),
            body = response.errorBody()?.string(),
            retryAfterHeader = response.headers()["Retry-After"],
        )
    }
    if (T::class == Unit::class) return Unit as T
    return response.body() ?: throw IOException("Empty 2xx body")
}
