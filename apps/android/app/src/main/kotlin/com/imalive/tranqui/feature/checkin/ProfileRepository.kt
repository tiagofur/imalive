package com.imalive.tranqui.feature.checkin

import com.imalive.tranqui.core.network.TranquiApi
import com.imalive.tranqui.core.network.apiCall
import com.imalive.tranqui.core.network.dto.ProfilePatch
import com.imalive.tranqui.core.network.dto.UserProfile
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

// The seam between the home and the profile API (GET/PATCH /v1/me). Failures
// surface typed (ApiError with the business code / IOException): the repo
// never swallows them — mapping them to UI states is the ViewModel's job.
interface ProfileRepository {
    suspend fun profile(): Profile

    /**
     * Partial update (PATCH /v1/me; issue #75 slice 4): the ritual step saves
     * name + check-in time + the device timezone. Returns the stored profile.
     */
    suspend fun update(patch: ProfilePatch): Profile
}

// Real read/update of /v1/me (issue #75 slices 2 and 4; replaces the F0
// stub). A degraded pauseUntil timestamp maps to null (no pause) instead of
// sinking the home; everything else is the contract's own wire shape.
@Singleton
class NetworkProfileRepository
    @Inject
    constructor(
        private val api: TranquiApi,
        private val json: Json,
    ) : ProfileRepository {
        override suspend fun profile(): Profile = profileOf(apiCall(json) { api.getProfile() })

        override suspend fun update(patch: ProfilePatch): Profile = profileOf(apiCall(json) { api.updateProfile(patch) })

        private fun profileOf(dto: UserProfile): Profile =
            Profile(
                streakDays = dto.streakDays,
                checkinTime = dto.checkinTime,
                pauseUntil = dto.pauseUntil?.let(::parseInstant),
            )

        private fun parseInstant(raw: String): Instant? = runCatching { Instant.parse(raw) }.getOrNull()
    }
