package com.imalive.tranqui.feature.checkin

import javax.inject.Inject
import javax.inject.Singleton

// The profile facts the home renders (contract v0.5.0: GET /v1/me carries
// streakDays; the pause state marks the streak as frozen, R-12).
data class Profile(
    val streakDays: Int,
    val streakFrozen: Boolean,
)

// The seam between the home and the profile API (GET /v1/me). The real
// Retrofit implementation lands with the auth flow (issue #75, onboarding):
// until there is a session, the stub keeps the UI honest and the ViewModel
// tested.
interface ProfileRepository {
    suspend fun profile(): Profile
}

// F0 stub: static values, the same shape the network client will deliver.
// Throwing is deliberately NOT simulated: the failure path (defaults without
// crashing) is exercised by the tests through a dedicated failing fake.
@Singleton
class StubProfileRepository
    @Inject
    constructor() : ProfileRepository {
        override suspend fun profile(): Profile = Profile(streakDays = 12, streakFrozen = false)
    }
