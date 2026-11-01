package com.imalive.tranqui.feature.checkin

import com.imalive.tranqui.core.network.TranquiApi
import com.imalive.tranqui.core.network.apiCall
import kotlinx.serialization.json.Json
import java.time.Instant
import javax.inject.Inject
import javax.inject.Singleton

// The profile facts the home renders (contract v0.5.0: GET /v1/me carries
// streakDays; the active pause R-11 is what freezes it, R-12).
data class Profile(
    val streakDays: Int,
    // "HH:MM" local slot of the daily check-in (contract pattern).
    val checkinTime: String,
    // The pause window, when there is one (R-11); null = no pause.
    val pauseUntil: Instant? = null,
)

// The seam between the home and the profile API (GET /v1/me). Failures
// surface typed (ApiError with the business code / IOException): the repo
// never swallows them — mapping them to UI states is the ViewModel's job.
interface ProfileRepository {
    suspend fun profile(): Profile
}

// Real read of GET /v1/me (issue #75 slice 2; replaces the F0 stub). A
// degraded pauseUntil timestamp maps to null (no pause) instead of sinking
// the home; everything else is the contract's own wire shape.
@Singleton
class NetworkProfileRepository
    @Inject
    constructor(
        private val api: TranquiApi,
        private val json: Json,
    ) : ProfileRepository {
        override suspend fun profile(): Profile {
            val dto = apiCall(json) { api.getProfile() }
            return Profile(
                streakDays = dto.streakDays,
                checkinTime = dto.checkinTime,
                pauseUntil = dto.pauseUntil?.let(::parseInstant),
            )
        }

        private fun parseInstant(raw: String): Instant? = runCatching { Instant.parse(raw) }.getOrNull()
    }
