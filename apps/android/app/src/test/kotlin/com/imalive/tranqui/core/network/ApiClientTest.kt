package com.imalive.tranqui.core.network

import com.imalive.tranqui.core.network.dto.OtpRequest
import com.imalive.tranqui.core.network.dto.OtpVerify
import com.imalive.tranqui.core.network.dto.Tokens
import com.imalive.tranqui.core.network.dto.UserProfile
import com.imalive.tranqui.core.session.SessionStore
import com.imalive.tranqui.core.session.StoredSession
import kotlinx.coroutines.runBlocking
import kotlinx.serialization.json.Json
import mockwebserver3.MockResponse
import mockwebserver3.MockWebServer
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.OkHttpClient
import org.junit.After
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Assert.assertThrows
import org.junit.Assert.assertTrue
import org.junit.Before
import org.junit.Test
import retrofit2.Retrofit
import retrofit2.converter.kotlinx.serialization.asConverterFactory
import java.io.IOException
import java.util.concurrent.TimeUnit

// Slice 1 of issue #75: the real client against a MockWebServer, with the
// contract's own examples as fixtures (ADR-0003). Request shapes, token
// attach, and the RFC 9457 error mapping live here.
class ApiClientTest {
    private lateinit var server: MockWebServer
    private lateinit var sessions: FakeSessionStore
    private lateinit var stack: TestStack
    private val json = ApiClientFactory.json()

    @Before
    fun setUp() {
        server = MockWebServer()
        server.start()
        sessions = FakeSessionStore()
        stack = TestStack(server, sessions, json)
    }

    @After
    fun tearDown() {
        server.close()
    }

    private fun api() = stack.api

    @Test
    fun `otp request posts the E-164 phone and accepts the 204`() =
        runBlocking {
            server.enqueue(MockResponse.Builder().code(204).build())

            val result = apiCall(json) { api().requestOtp(OtpRequest(phone = "+5215512345678")) }

            assertEquals(Unit, result)
            val recorded = server.takeRequest()
            assertEquals("POST", recorded.method)
            assertEquals("/v1/auth/otp/request", recorded.target)
            assertEquals("""{"phone":"+5215512345678"}""", recorded.body!!.utf8())
            // Auth endpoints are public: no Authorization header even with a session.
            assertNull(recorded.headers["Authorization"])
        }

    @Test
    fun `otp request maps 429 Retry-After into the ApiError`() =
        runBlocking {
            server.enqueue(
                jsonResponse(
                    429,
                    // contracts/openapi.yaml, responses.TooManyRequests example.
                    """
                    {
                      "type": "https://api.tranqui.app/problems/rate-limited",
                      "title": "Too Many Requests",
                      "status": 429,
                      "detail": "Too many requests; slow down and retry later.",
                      "code": "RATE_LIMITED",
                      "instance": "/v1/auth/otp/request"
                    }
                    """.trimIndent(),
                    "Retry-After" to "60",
                ),
            )

            val error =
                assertThrows(ApiError::class.java) {
                    runBlocking { apiCall(json) { api().requestOtp(OtpRequest(phone = "+5215512345678")) } }
                }

            assertEquals(ApiErrorCode.RATE_LIMITED, error.code)
            assertEquals(429, error.httpStatus)
            assertEquals(60, error.retryAfterSeconds)
        }

    @Test
    fun `otp verify parses the token pair from the contract example`() =
        runBlocking {
            server.enqueue(jsonResponse(200, TOKENS_EXAMPLE))

            val tokens = apiCall(json) { api().verifyOtp(OtpVerify(phone = "+5215512345678", code = "482913")) }

            assertEquals("eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.example", tokens.accessToken)
            assertEquals("7f3a9c2e5b1d4a6f8e0c3b5d7a9e1f2a4", tokens.refreshToken)
            assertEquals(900, tokens.expiresInSeconds)
            val recorded = server.takeRequest()
            assertEquals("/v1/auth/otp/verify", recorded.target)
            assertEquals("""{"phone":"+5215512345678","code":"482913"}""", recorded.body!!.utf8())
        }

    @Test
    fun `otp verify maps the OTP_EXPIRED problem to ApiError`() =
        runBlocking {
            server.enqueue(
                jsonResponse(
                    400,
                    // contracts/openapi.yaml, POST /auth/otp/verify 400 "expired" example.
                    """
                    {
                      "type": "https://api.tranqui.app/problems/otp-expired",
                      "title": "Code expired",
                      "status": 400,
                      "detail": "The code expired; request a new one.",
                      "code": "OTP_EXPIRED",
                      "instance": "/v1/auth/otp/verify"
                    }
                    """.trimIndent(),
                ),
            )

            val error =
                assertThrows(ApiError::class.java) {
                    runBlocking {
                        val call = api().verifyOtp(OtpVerify(phone = "+5215512345678", code = "111111"))
                        apiCall<Tokens>(json) { call }
                    }
                }

            assertEquals(ApiErrorCode.OTP_EXPIRED, error.code)
            assertEquals(400, error.httpStatus)
            assertNull(error.retryAfterSeconds)
        }

    @Test
    fun `protected calls carry the bearer token of the session`() =
        runBlocking {
            sessions.save(
                StoredSession(
                    accessToken = "access-token-1",
                    refreshToken = "refresh-token-1",
                    expiresAtEpochSeconds = 0L,
                ),
            )
            server.enqueue(jsonResponse(200, PROFILE_EXAMPLE))

            val profile = apiCall(json) { api().getProfile() }

            assertEquals("Ana", profile.name)
            assertEquals(12, profile.streakDays)
            assertEquals("Bearer access-token-1", server.takeRequest().headers["Authorization"])
        }

    @Test
    fun `without a session protected calls go out anonymous and 401 maps to ApiError`() =
        runBlocking {
            server.enqueue(jsonResponse(401, UNAUTHORIZED_PROBLEM))

            val error =
                assertThrows(ApiError::class.java) {
                    runBlocking { apiCall<UserProfile>(json) { api().getProfile() } }
                }

            assertEquals(ApiErrorCode.UNAUTHORIZED, error.code)
            assertNull(server.takeRequest().headers["Authorization"])
        }

    @Test
    fun `an offline call surfaces as IOException, never as ApiError`() {
        server.close() // nothing listening: connection refused

        val thrown =
            assertThrows(Exception::class.java) {
                runBlocking { apiCall(json) { api().requestOtp(OtpRequest(phone = "+5215512345678")) } }
            }

        assertTrue("expected IOException but got $thrown", thrown is IOException)
    }

    private fun jsonResponse(
        code: Int,
        body: String,
        vararg headers: Pair<String, String>,
    ): MockResponse {
        val builder =
            MockResponse
                .Builder()
                .code(code)
                .setHeader("Content-Type", "application/json")
        headers.forEach { (name, value) -> builder.setHeader(name, value) }
        return builder.body(body).build()
    }

    // Assembles the production wiring via ApiClientFactory semantics but with
    // the MockWebServer base URL and a guard timeout. The refresh API is the
    // same stack minus the authenticator, so a 401 of /auth/refresh cannot
    // recurse into authenticate().
    private class TestStack(
        private val server: MockWebServer,
        val sessionsStore: SessionStore,
        private val json: Json,
    ) {
        val api: TranquiApi by lazy { createApi() }

        private fun createApi(): TranquiApi {
            val refreshApi: TranquiApi by lazy { createApiWithoutAuthenticator() }
            val client =
                OkHttpClient
                    .Builder()
                    .addInterceptor(AuthInterceptor(sessionsStore))
                    .authenticator(TokenAuthenticator(sessionsStore, dagger.Lazy<TranquiApi> { refreshApi }))
                    .addInterceptor(ApiClientFactory.loggingInterceptor(debug = false))
                    .callTimeout(10, TimeUnit.SECONDS)
                    .build()
            return retrofit(client).create(TranquiApi::class.java)
        }

        private fun createApiWithoutAuthenticator(): TranquiApi {
            val client =
                OkHttpClient
                    .Builder()
                    .addInterceptor(AuthInterceptor(sessionsStore))
                    .addInterceptor(ApiClientFactory.loggingInterceptor(debug = false))
                    .callTimeout(10, TimeUnit.SECONDS)
                    .build()
            return retrofit(client).create(TranquiApi::class.java)
        }

        private fun retrofit(client: OkHttpClient): Retrofit =
            Retrofit
                .Builder()
                .baseUrl(server.url("/v1/"))
                .client(client)
                .addConverterFactory(json.asConverterFactory("application/json".toMediaType()))
                .build()
    }

    internal companion object {
        // contracts/openapi.yaml examples, verbatim fixtures.
        internal val UNAUTHORIZED_PROBLEM: String =
            """
            {
              "type": "https://api.tranqui.app/problems/unauthorized",
              "title": "Unauthorized",
              "status": 401,
              "detail": "Missing, expired or invalid credentials.",
              "code": "UNAUTHORIZED",
              "instance": "/v1/me"
            }
            """.trimIndent()

        internal val TOKENS_EXAMPLE: String =
            """
            {
              "accessToken": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.example",
              "refreshToken": "7f3a9c2e5b1d4a6f8e0c3b5d7a9e1f2a4",
              "expiresInSeconds": 900
            }
            """.trimIndent()

        internal val PROFILE_EXAMPLE: String =
            """
            {
              "id": "01934f2e-7a1b-7c3d-9e4f-1a2b3c4d5e6f",
              "name": "Ana",
              "phone": "+5215512345678",
              "checkinTime": "21:00",
              "timezone": "America/Mexico_City",
              "escalationWindowMinutes": 60,
              "streakDays": 12
            }
            """.trimIndent()
    }
}
