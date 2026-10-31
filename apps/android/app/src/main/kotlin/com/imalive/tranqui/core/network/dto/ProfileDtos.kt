package com.imalive.tranqui.core.network.dto

import kotlinx.serialization.Serializable

// Authenticated user profile (contract v0.5.0, GET /me). Minimum data by
// design (R-02): no email, address, notes or avatar.
@Serializable
data class UserProfile(
    val id: String,
    val name: String,
    val phone: String,
    // "HH:MM" local slot of the daily check-in.
    val checkinTime: String,
    // IANA timezone id, e.g. America/Mexico_City.
    val timezone: String,
    val escalationWindowMinutes: Int,
    val streakDays: Int,
)

// Partial profile update (PATCH /me); omitted fields keep their values.
// encodeDefaults = false in the shared Json keeps unset nulls off the wire.
@Serializable
data class ProfilePatch(
    val name: String? = null,
    val checkinTime: String? = null,
    val timezone: String? = null,
    val escalationWindowMinutes: Int? = null,
)
