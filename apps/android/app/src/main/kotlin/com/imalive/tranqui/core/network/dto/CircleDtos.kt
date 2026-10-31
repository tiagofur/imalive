package com.imalive.tranqui.core.network.dto

import kotlinx.serialization.Serializable

// Trusted-circle contact (R-03): name + phone, nothing else.
@Serializable
data class Contact(
    val id: String,
    val name: String,
    val phone: String,
    // RFC 3339, UTC.
    val createdAt: String,
)

// New contact payload (POST /contacts); the backend sends the welcome
// message to the contact.
@Serializable
data class ContactCreate(
    val name: String,
    val phone: String,
)

// The circle is product-bounded to 5 contacts; no pagination.
@Serializable
data class ContactList(
    val contacts: List<Contact>,
)

// Today's recorded "I'm OK" signal (R-04).
@Serializable
data class Checkin(
    val date: String,
    val checkedInAt: String,
    val replayed: Boolean,
)
