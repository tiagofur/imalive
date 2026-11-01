package com.imalive.tranqui.feature.checkin

import com.imalive.tranqui.core.network.ApiClientFactory
import com.imalive.tranqui.core.network.ApiClientTest
import com.imalive.tranqui.core.network.ApiError
import com.imalive.tranqui.core.network.ApiErrorCode
import com.imalive.tranqui.core.network.AuthInterceptor
import com.imalive.tranqui.core.network.FakeSessionStore
import com.imalive.tranqui.core.network.TranquiApi
import com.imalive.tranqui.core.session.StoredSession
import kotlinx.coroutines.runBlocking
import kotlinx.serialization.json.Json
import mockwebserver3.MockResponse
import mockwebserver3.MockWebServer
import okhttp3.OkHttpClient
import org.junit.After
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Assert.assertThrows
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
    }
}
