package com.imalive.tranqui.feature.checkin

import com.imalive.tranqui.core.network.ApiClientFactory
import com.imalive.tranqui.core.network.ApiClientTest
import com.imalive.tranqui.core.network.ApiError
import com.imalive.tranqui.core.network.ApiErrorCode
import com.imalive.tranqui.core.network.AuthInterceptor
import com.imalive.tranqui.core.network.FakeSessionStore
import com.imalive.tranqui.core.network.TranquiApi
import com.imalive.tranqui.core.network.dto.ProfilePatch
import com.imalive.tranqui.core.session.StoredSession
import kotlinx.coroutines.flow.first
import kotlinx.coroutines.runBlocking
import kotlinx.serialization.json.Json
import mockwebserver3.MockResponse
import mockwebserver3.MockWebServer
import okhttp3.OkHttpClient
import org.junit.After
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Assert.assertThrows
import org.junit.Assert.assertTrue
import org.junit.Before
import org.junit.Test
import java.time.Instant

// Slice 2 of issue #75: the real repositories against a MockWebServer, with
// the contract's own examples as fixtures (ADR-0003 — shared verbatim with
// ApiClientTest). The repos surface typed failures (ApiError / IOException);
// turning them into UI states is the ViewModels' job.
class NetworkRepositoriesTest {
    private lateinit var server: MockWebServer
    private lateinit var sessions: FakeSessionStore
    private lateinit var profiles: NetworkProfileRepository
    private lateinit var auth: NetworkAuthRepository
    private lateinit var contacts: NetworkContactsRepository
    private val json: Json = ApiClientFactory.json()

    @Before
    fun setUp() {
        server = MockWebServer()
        server.start()
        sessions =
            FakeSessionStore(
                StoredSession(
                    accessToken = "access-token-1",
                    refreshToken = "refresh-token-1",
                    expiresAtEpochSeconds = 0L,
                ),
            )
        profiles = NetworkProfileRepository(api(), json)
        auth = NetworkAuthRepository(api(), json, sessions)
        contacts = NetworkContactsRepository(api(), json)
    }

    @After
    fun tearDown() {
        server.close()
    }

    // --- profile (GET /v1/me) ---

    @Test
    fun `profile reads GET me with the bearer and parses the contract example`() =
        runBlocking {
            server.enqueue(jsonResponse(200, ApiClientTest.PROFILE_EXAMPLE))

            val profile = profiles.profile()

            assertEquals(12, profile.streakDays)
            assertEquals("21:00", profile.checkinTime)
            assertNull(profile.pauseUntil)
            val recorded = server.takeRequest()
            assertEquals("/v1/me", recorded.target)
            assertEquals("Bearer access-token-1", recorded.headers["Authorization"])
        }

    @Test
    fun `profile maps the pause window of the schema example into an instant`() =
        runBlocking {
            server.enqueue(jsonResponse(200, PROFILE_PAUSED_EXAMPLE))

            val profile = profiles.profile()

            assertEquals(Instant.parse("2026-11-15T06:00:00Z"), profile.pauseUntil)
            assertEquals(12, profile.streakDays)
        }

    @Test
    fun `profile survives a degraded pauseUntil as no pause`() =
        runBlocking {
            server.enqueue(jsonResponse(200, PROFILE_DEGRADED_PAUSE_EXAMPLE))

            val profile = profiles.profile()

            assertEquals(12, profile.streakDays)
            assertNull(profile.pauseUntil)
        }

    @Test
    fun `update posts the ritual patch with only the fields the onboarding sets`() =
        runBlocking {
            server.enqueue(jsonResponse(200, PROFILE_PATCHED_EXAMPLE))

            profiles.update(
                ProfilePatch(name = "Ana", checkinTime = "21:00", timezone = "America/Mexico_City"),
            )

            val recorded = server.takeRequest()
            assertEquals("PATCH", recorded.method)
            assertEquals("/v1/me", recorded.target)
            assertEquals(
                """{"name":"Ana","checkinTime":"21:00","timezone":"America/Mexico_City"}""",
                recorded.body!!.utf8(),
            )
        }

    @Test
    fun `update survives the backend body that omits streakDays until the backend honors the contract`() =
        runBlocking {
            // The live backend answers PATCH with profileOf(user), without
            // streakDays (contract deviation, tracked backend-side); the
            // ritual must not die decoding a body it does not consume.
            server.enqueue(
                jsonResponse(
                    200,
                    """
                    {
                      "id": "01934f2e-7a1b-7c3d-9e4f-1a2b3c4d5e6f",
                      "name": "Ana",
                      "phone": "+5215512345678",
                      "checkinTime": "21:00",
                      "timezone": "America/Mexico_City",
                      "escalationWindowMinutes": 60
                    }
                    """.trimIndent(),
                ),
            )

            profiles.update(ProfilePatch(name = "Ana", checkinTime = "21:00", timezone = "America/Mexico_City"))

            assertEquals("/v1/me", server.takeRequest().target)
        }

    @Test
    fun `profile surfaces the 401 problem as a typed ApiError, never swallowed`() =
        runBlocking {
            server.enqueue(jsonResponse(401, ApiClientTest.UNAUTHORIZED_PROBLEM))

            val error =
                assertThrows(ApiError::class.java) {
                    runBlocking { profiles.profile() }
                }

            assertEquals(ApiErrorCode.UNAUTHORIZED, error.code)
            assertEquals(401, error.httpStatus)
        }

    // --- auth (POST /auth/otp/*) ---

    @Test
    fun `otp request posts the phone and accepts the 204 anonymously`() =
        runBlocking {
            server.enqueue(jsonResponse(204, ""))

            auth.requestOtp("+5215512345678")

            val recorded = server.takeRequest()
            assertEquals("/v1/auth/otp/request", recorded.target)
            assertEquals("""{"phone":"+5215512345678"}""", recorded.body!!.utf8())
            assertNull(recorded.headers["Authorization"])
        }

    @Test
    fun `otp verify persists the token pair and the phone in the session`() =
        runBlocking {
            server.enqueue(jsonResponse(200, ApiClientTest.TOKENS_EXAMPLE))
            val before = Instant.now().epochSecond

            auth.verifyOtp(phone = "+5215512345678", code = "482913")

            val saved = sessions.session.first()
            assertEquals("eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.example", saved!!.accessToken)
            assertEquals("7f3a9c2e5b1d4a6f8e0c3b5d7a9e1f2a4", saved.refreshToken)
            assertEquals("+5215512345678", saved.userPhone)
            // now + expiresInSeconds, within the second the test took.
            val after = Instant.now().epochSecond
            assertTrue(saved.expiresAtEpochSeconds in before + 900L..after + 900L)
        }

    @Test
    fun `otp verify with an expired code surfaces OTP_EXPIRED and keeps the stored session`() =
        runBlocking {
            server.enqueue(jsonResponse(400, OTP_EXPIRED_PROBLEM))

            val error =
                assertThrows(ApiError::class.java) {
                    runBlocking { auth.verifyOtp(phone = "+5215512345678", code = "111111") }
                }

            assertEquals(ApiErrorCode.OTP_EXPIRED, error.code)
            assertEquals("access-token-1", sessions.session.first()!!.accessToken)
        }

    // --- contacts (GET/POST /v1/contacts) ---

    @Test
    fun `contacts lists the circle from the contract example`() =
        runBlocking {
            server.enqueue(jsonResponse(200, CONTACTS_LIST_EXAMPLE))

            val circle = contacts.contacts()

            assertEquals(2, circle.size)
            assertEquals("María", circle[0].name)
            assertEquals("+5215598765432", circle[0].phone)
            assertEquals("Jorge", circle[1].name)
            assertEquals("/v1/contacts", server.takeRequest().target)
        }

    @Test
    fun `create posts name and phone and returns the 201 contact`() =
        runBlocking {
            server.enqueue(jsonResponse(201, CONTACT_EXAMPLE))

            val contact = contacts.create(name = "María", phone = "+5215598765432")

            assertEquals("01934f2f-8a2c-7d4e-af5a-2b3c4d5e6f7a", contact.id)
            assertEquals("María", contact.name)
            assertEquals("+5215598765432", contact.phone)
            assertEquals(
                """{"name":"María","phone":"+5215598765432"}""",
                server.takeRequest().body!!.utf8(),
            )
        }

    @Test
    fun `create maps CONTACT_LIMIT_REACHED to a typed ApiError`() =
        runBlocking {
            server.enqueue(jsonResponse(422, CONTACT_LIMIT_PROBLEM))

            val error =
                assertThrows(ApiError::class.java) {
                    runBlocking { contacts.create(name = "Jorge", phone = "+5215511223344") }
                }

            assertEquals(ApiErrorCode.CONTACT_LIMIT_REACHED, error.code)
            assertEquals(422, error.httpStatus)
        }

    // Production-shaped stack for the repo under test: bearer attach via the
    // real AuthInterceptor, MockWebServer as base URL. No authenticator here:
    // the refresh paths belong to TokenAuthenticatorTest (slice 1).
    private fun api(): TranquiApi {
        val client =
            OkHttpClient
                .Builder()
                .addInterceptor(AuthInterceptor(sessions))
                .build()
        return ApiClientFactory
            .retrofit(server.url("/v1/").toString(), client, json)
            .create(TranquiApi::class.java)
    }

    private fun jsonResponse(
        code: Int,
        body: String,
    ): MockResponse =
        MockResponse
            .Builder()
            .code(code)
            .setHeader("Content-Type", "application/json")
            .body(body)
            .build()

    private companion object {
        // contracts/openapi.yaml: UserProfile schema example (pauseUntil,
        // R-11) and the same profile with a garbage timestamp.
        private val PROFILE_PAUSED_EXAMPLE: String =
            """
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
            """.trimIndent()

        private val PROFILE_DEGRADED_PAUSE_EXAMPLE: String =
            """
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
            """.trimIndent()

        // contracts/openapi.yaml, POST /auth/otp/verify 400 "expired" example.
        private val OTP_EXPIRED_PROBLEM: String =
            """
            {
              "type": "https://api.tranqui.app/problems/otp-expired",
              "title": "Code expired",
              "status": 400,
              "detail": "The code expired; request a new one.",
              "code": "OTP_EXPIRED",
              "instance": "/v1/auth/otp/verify"
            }
            """.trimIndent()

        // contracts/openapi.yaml, PATCH /me 200 example (the ritual fields).
        private val PROFILE_PATCHED_EXAMPLE: String =
            """
            {
              "id": "01934f2e-7a1b-7c3d-9e4f-1a2b3c4d5e6f",
              "name": "Ana",
              "phone": "+5215512345678",
              "checkinTime": "21:00",
              "timezone": "America/Mexico_City",
              "escalationWindowMinutes": 60,
              "streakDays": 0
            }
            """.trimIndent()

        // contracts/openapi.yaml, GET /contacts 200 example.
        private val CONTACTS_LIST_EXAMPLE: String =
            """
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
            """.trimIndent()

        // contracts/openapi.yaml, POST /contacts 201 example.
        private val CONTACT_EXAMPLE: String =
            """
            {
              "id": "01934f2f-8a2c-7d4e-af5a-2b3c4d5e6f7a",
              "name": "María",
              "phone": "+5215598765432",
              "createdAt": "2026-10-20T15:30:00Z"
            }
            """.trimIndent()

        // contracts/openapi.yaml, POST /contacts 422 example (verbatim,
        // Spanish detail included).
        private val CONTACT_LIMIT_PROBLEM: String =
            """
            {
              "type": "https://api.tranqui.app/problems/contact-limit-reached",
              "title": "Límite de contactos alcanzado",
              "status": 422,
              "detail": "El plan gratuito permite hasta 1 contacto. Actualiza a Premium.",
              "code": "CONTACT_LIMIT_REACHED",
              "instance": "/v1/contacts"
            }
            """.trimIndent()
    }
}
