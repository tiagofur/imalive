package com.imalive.tranqui.core.network

import com.imalive.tranqui.core.network.dto.RefreshRequest
import com.imalive.tranqui.core.session.SessionStore
import com.imalive.tranqui.core.session.StoredSession
import dagger.Lazy
import kotlinx.coroutines.runBlocking
import kotlinx.coroutines.sync.Mutex
import kotlinx.coroutines.sync.withLock
import okhttp3.Authenticator
import okhttp3.Request
import okhttp3.Response
import okhttp3.Route
import java.io.IOException
import java.time.Instant
import javax.inject.Inject
import javax.inject.Singleton

/**
 * Rotates an expired session exactly once per burst: concurrent 401s funnel
 * through a single-flight mutex, so only one POST /auth/refresh goes on the
 * wire and the rest reuse the rotated tokens (plan issue #75: single-flight).
 *
 * Outcome per attempt:
 * - Refresh ok → retry the original request once with the new token.
 * - Refresh rejected (401/problem) → session wiped; the gate sends the user
 *   back to onboarding. The original request fails.
 * - Refresh unreachable (IOException) → session kept (offline must not log
 *   the user out); only the original request is given up.
 */
@Singleton
class TokenAuthenticator
    @Inject
    constructor(
        private val sessions: SessionStore,
        // Lazy breaks the DI cycle: the API's OkHttp client is the one that
        // holds this authenticator.
        private val api: Lazy<TranquiApi>,
    ) : Authenticator {
        private val refreshMutex = Mutex()

        override fun authenticate(
            route: Route?,
            response: Response,
        ): Request? {
            val path = response.request.url.encodedPath
            // A 401 of the auth endpoints themselves is terminal: never refresh
            // because of the refresh call (recursion guard) nor of a bad OTP.
            // Only one retry per original request either: a second 401 with a
            // fresh token means the session is dead server-side.
            val refreshable = !path.contains(AUTH_PATH_SEGMENT) && responseCount(response) < 2
            if (!refreshable) return null
            return retryWithRotatedSession(response, staleAccess(response))
        }

        /** The stale access token of the failed request, if it carried one. */
        private fun staleAccess(response: Response): String? {
            return response.request.header("Authorization")?.removePrefix(BEARER_PREFIX)
        }

        /** The single-flight rotate-and-retry, or null when the session is dead. */
        private fun retryWithRotatedSession(
            response: Response,
            staleAccess: String?,
        ): Request? {
            // OkHttp's Authenticator is synchronous; the single-flight refresh
            // blocks this request's thread only, on the 401 path (never on the
            // happy path).
            val session =
                runBlocking {
                    refreshMutex.withLock {
                        val current = sessions.cachedSession()
                        when {
                            current == null -> null
                            // Another thread already rotated while we waited.
                            staleAccess != null && current.accessToken != staleAccess -> current
                            else -> rotate(current)
                        }
                    }
                }
            return session?.let { refreshed ->
                response.request
                    .newBuilder()
                    .header("Authorization", BEARER_PREFIX + refreshed.accessToken)
                    .build()
            }
        }

        /** Calls POST /auth/refresh and persists the rotation, or gives up. */
        private suspend fun rotate(current: StoredSession): StoredSession? =
            try {
                val response = api.get().refreshSession(RefreshRequest(refreshToken = current.refreshToken))
                if (!response.isSuccessful) {
                    sessions.clear() // rejected refresh: the session is invalid
                    null
                } else {
                    val tokens = response.body() ?: return null
                    val rotated =
                        current.copy(
                            accessToken = tokens.accessToken,
                            refreshToken = tokens.refreshToken,
                            expiresAtEpochSeconds = Instant.now().epochSecond + tokens.expiresInSeconds,
                        )
                    sessions.save(rotated)
                    rotated
                }
            } catch (_: IOException) {
                null // transient: keep the session, give up just this request
            } catch (_: Exception) {
                sessions.clear() // undecodable answer: treat as dead session
                null
            }

        private fun responseCount(response: Response): Int {
            var count = 1
            var prior = response.priorResponse
            while (prior != null) {
                count++
                prior = prior.priorResponse
            }
            return count
        }

        private companion object {
            const val AUTH_PATH_SEGMENT = "/auth/"
            const val BEARER_PREFIX = "Bearer "
        }
    }
