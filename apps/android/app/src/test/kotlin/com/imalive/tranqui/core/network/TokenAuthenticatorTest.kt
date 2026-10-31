package com.imalive.tranqui.core.network

import com.imalive.tranqui.core.network.dto.UserProfile
import com.imalive.tranqui.core.session.StoredSession
import kotlinx.coroutines.runBlocking
import mockwebserver3.Dispatcher
import mockwebserver3.MockResponse
import mockwebserver3.MockWebServer
import mockwebserver3.RecordedRequest
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
import java.util.concurrent.CountDownLatch
import java.util.concurrent.Executors
import java.util.concurrent.TimeUnit
import java.util.concurrent.atomic.AtomicInteger

// The refresh single-flight of issue #75: concurrent 401s must produce
// exactly ONE POST /auth/refresh; a rejected refresh must wipe the session;
// an unreachable refresh must NOT (offline never logs the user out). The
// stack mirrors production: ONE OkHttp client, the refresh call included,
// with the /auth/ path guard preventing recursion.
class TokenAuthenticatorTest {
    private lateinit var server: MockWebServer
    private lateinit var sessions: FakeSessionStore
    private val json = ApiClientFactory.json()

    // Lazy: built on first use, once the dispatcher and the server are up.
    private val api: TranquiApi by lazy {
        val client =
            OkHttpClient
                .Builder()
                .addInterceptor(AuthInterceptor(sessions))
                .authenticator(TokenAuthenticator(sessions, dagger.Lazy<TranquiApi> { api }))
                .addInterceptor(ApiClientFactory.loggingInterceptor(debug = false))
                .callTimeout(10, TimeUnit.SECONDS)
                .build()
        Retrofit
            .Builder()
            .baseUrl(server.url("/v1/"))
            .client(client)
            .addConverterFactory(json.asConverterFactory("application/json".toMediaType()))
            .build()
            .create(TranquiApi::class.java)
    }

    @Before
    fun setUp() {
        server = MockWebServer()
        sessions = FakeSessionStore()
        runBlocking { sessions.save(session()) }
    }

    @After
    fun tearDown() {
        server.close()
    }

    private fun session(accessToken: String = "stale-access-token"): StoredSession =
        StoredSession(
            accessToken = accessToken,
            refreshToken = "refresh-token-1",
            expiresAtEpochSeconds = 0L,
        )

    private fun ok(body: String): MockResponse =
        MockResponse
            .Builder()
            .code(200)
            .setHeader("Content-Type", "application/json")
            .body(body)
            .build()

    private fun problem(
        code: Int,
        body: String,
    ): MockResponse =
        MockResponse
            .Builder()
            .code(code)
            .setHeader("Content-Type", "application/problem+json")
            .body(body)
            .build()

    @Test
    fun `four concurrent 401s produce exactly one refresh and all retry with the rotated token`() {
        val refreshCount = AtomicInteger(0)
        val profileServed401 = AtomicInteger(0)
        server.dispatcher =
            object : Dispatcher() {
                override fun dispatch(request: RecordedRequest): MockResponse =
                    when {
                        // The /auth/ path guard must keep this at exactly one call.
                        request.target.endsWith("/auth/refresh") -> {
                            refreshCount.incrementAndGet()
                            ok(ApiClientTest.TOKENS_EXAMPLE)
                        }
                        // The profile answers 401 until the rotation lands.
                        request.target.endsWith("/me") ->
                            if (refreshCount.get() == 0) {
                                profileServed401.incrementAndGet()
                                problem(401, ApiClientTest.UNAUTHORIZED_PROBLEM)
                            } else {
                                ok(ApiClientTest.PROFILE_EXAMPLE)
                            }
                        else -> throw AssertionError("unexpected call to ${request.target}")
                    }
            }
        server.start()

        val concurrency = 4
        val allReady = CountDownLatch(concurrency)
        val pool = Executors.newFixedThreadPool(concurrency)
        try {
            val results =
                (1..concurrency).map {
                    pool.submit<String> {
                        runBlocking {
                            allReady.countDown()
                            allReady.await()
                            apiCall(json) { api.getProfile() }.name
                        }
                    }
                }
            val names = results.map { it.get(15, TimeUnit.SECONDS) }

            assertEquals(listOf("Ana", "Ana", "Ana", "Ana"), names.sorted())
            // The heart of the test: single flight.
            assertEquals(1, refreshCount.get())
            assertTrue("expected the 401 path to be exercised", profileServed401.get() >= 1)
            // The session now holds the rotated pair.
            assertEquals("eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.example", sessions.cachedSession()?.accessToken)
        } finally {
            pool.shutdownNow()
        }
    }

    @Test
    fun `a rejected refresh wipes the session and the original call fails`() {
        server.dispatcher =
            object : Dispatcher() {
                override fun dispatch(request: RecordedRequest): MockResponse =
                    when {
                        request.target.endsWith("/auth/refresh") -> problem(401, ApiClientTest.UNAUTHORIZED_PROBLEM)
                        request.target.endsWith("/me") -> problem(401, ApiClientTest.UNAUTHORIZED_PROBLEM)
                        else -> throw AssertionError("unexpected call to ${request.target}")
                    }
            }
        server.start()

        val error =
            assertThrows(ApiError::class.java) {
                runBlocking { apiCall<UserProfile>(json) { api.getProfile() } }
            }

        assertEquals(ApiErrorCode.UNAUTHORIZED, error.code)
        assertEquals(401, error.httpStatus)
        assertEquals("the rejected refresh must invalidate the session", 1, sessions.clearCount)
        assertNull(sessions.cachedSession())
    }

    @Test
    fun `an unreachable refresh keeps the session and gives up only the request`() {
        server.dispatcher =
            object : Dispatcher() {
                override fun dispatch(request: RecordedRequest): MockResponse =
                    when {
                        // The refresh exchange gets its connection cut: IOException.
                        request.target.endsWith("/auth/refresh") ->
                            MockResponse.Builder().onResponseStart(mockwebserver3.SocketEffect.CloseSocket()).build()
                        request.target.endsWith("/me") -> problem(401, ApiClientTest.UNAUTHORIZED_PROBLEM)
                        else -> throw AssertionError("unexpected call to ${request.target}")
                    }
            }
        server.start()

        val error =
            assertThrows(ApiError::class.java) {
                runBlocking { apiCall<UserProfile>(json) { api.getProfile() } }
            }

        // The original request fails with its own 401, but the outage does NOT
        // log the user out: the session survives for the next attempt.
        assertEquals(ApiErrorCode.UNAUTHORIZED, error.code)
        assertEquals("an unreachable refresh must not clear the session", 0, sessions.clearCount)
        assertEquals("stale-access-token", sessions.cachedSession()?.accessToken)
    }

    @Test
    fun `the refresh call never re-enters the authenticator`() {
        val refreshAttempts = AtomicInteger(0)
        server.dispatcher =
            object : Dispatcher() {
                override fun dispatch(request: RecordedRequest): MockResponse =
                    when {
                        request.target.endsWith("/auth/refresh") -> {
                            refreshAttempts.incrementAndGet()
                            problem(401, ApiClientTest.UNAUTHORIZED_PROBLEM)
                        }
                        request.target.endsWith("/me") -> problem(401, ApiClientTest.UNAUTHORIZED_PROBLEM)
                        else -> throw AssertionError("unexpected call to ${request.target}")
                    }
            }
        server.start()

        assertThrows(ApiError::class.java) {
            runBlocking { apiCall<UserProfile>(json) { api.getProfile() } }
        }

        // One /me 401, one refresh 401, and NOT a second refresh from the
        // recursion: the guard stops the chain.
        assertEquals(1, refreshAttempts.get())
        assertEquals(2, server.requestCount)
    }
}
