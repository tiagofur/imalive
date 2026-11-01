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
    // Active pause (R-11): escalation frozen until this instant (RFC 3339
    // UTC; null/absent = no pause). Optional in the contract, so the default
    // keeps payloads without it decodable.
    val pauseUntil: String? = null,
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
