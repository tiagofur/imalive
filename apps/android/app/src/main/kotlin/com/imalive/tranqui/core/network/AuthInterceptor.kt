package com.imalive.tranqui.core.network

import com.imalive.tranqui.core.session.SessionStore
import okhttp3.Interceptor
import okhttp3.Response
import javax.inject.Inject
import javax.inject.Singleton

/**
 * Attaches the current access token as `Authorization: Bearer …` to every
 * protected call. Auth endpoints (public per contract, `security: []`) and
 * calls without a session go through untouched. The header is redacted in
 * the logging interceptor, so the token never reaches a log.
 */
@Singleton
class AuthInterceptor
    @Inject
    constructor(
        private val sessions: SessionStore,
    ) : Interceptor {
        override fun intercept(chain: Interceptor.Chain): Response {
            val request = chain.request()
            val onAuthEndpoint = request.url.encodedPath.contains(AUTH_PATH_SEGMENT)
            val token = if (onAuthEndpoint) null else sessions.cachedSession()?.accessToken
            if (token == null) return chain.proceed(request)
            return chain.proceed(
                request.newBuilder().header("Authorization", "Bearer $token").build(),
            )
        }

        private companion object {
            const val AUTH_PATH_SEGMENT = "/auth/"
        }
    }
