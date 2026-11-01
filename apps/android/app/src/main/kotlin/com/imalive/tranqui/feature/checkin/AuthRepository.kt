package com.imalive.tranqui.feature.checkin

import com.imalive.tranqui.core.network.TranquiApi
import com.imalive.tranqui.core.network.apiCall
import com.imalive.tranqui.core.network.dto.OtpRequest
import com.imalive.tranqui.core.network.dto.OtpVerify
import com.imalive.tranqui.core.session.SessionStore
import com.imalive.tranqui.core.session.StoredSession
import kotlinx.serialization.json.Json
import java.time.Instant
import javax.inject.Inject
import javax.inject.Singleton

// The auth seam of the onboarding (R-01, issue #75): request the OTP for a
// phone and verify it into a persisted session. Rotating an expired session
// is NOT here — it lives in the OkHttp authenticator (slice 1), so every
// call of the app refreshes transparently. Failures surface typed
// (ApiError / IOException); the copy per business code is UI's job.
interface AuthRepository {
    suspend fun requestOtp(phone: String)

    /** Verifies the code and persists the token pair; the session gate then lets the user through. */
    suspend fun verifyOtp(
        phone: String,
        code: String,
    )
}

@Singleton
class NetworkAuthRepository
    @Inject
    constructor(
        private val api: TranquiApi,
        private val json: Json,
        private val sessions: SessionStore,
    ) : AuthRepository {
        override suspend fun requestOtp(phone: String) {
            apiCall(json) { api.requestOtp(OtpRequest(phone)) } // 204: nothing to carry back
        }

        override suspend fun verifyOtp(
            phone: String,
            code: String,
        ) {
            val tokens = apiCall(json) { api.verifyOtp(OtpVerify(phone = phone, code = code)) }
            sessions.save(
                StoredSession(
                    accessToken = tokens.accessToken,
                    refreshToken = tokens.refreshToken,
                    expiresAtEpochSeconds = Instant.now().epochSecond + tokens.expiresInSeconds,
                    userPhone = phone,
                ),
            )
        }
    }
