package com.imalive.tranqui.feature.checkin

import com.imalive.tranqui.core.network.TranquiApi
import com.imalive.tranqui.core.network.apiCall
import com.imalive.tranqui.core.network.dto.Contact
import com.imalive.tranqui.core.network.dto.ContactCreate
import kotlinx.serialization.json.Json
import javax.inject.Inject
import javax.inject.Singleton

// The trusted circle (R-03): adding a contact makes the backend send the
// WhatsApp/SMS welcome — the contact never registers or installs anything.
// The wire DTO doubles as the domain type: pure data (name + phone + ids),
// no product rules (the 1..5 quota is enforced server-side and arrives as
// CONTACT_LIMIT_REACHED). Failures surface typed, never swallowed.
interface ContactsRepository {
    suspend fun contacts(): List<Contact>

    suspend fun create(
        name: String,
        phone: String,
    ): Contact
}

@Singleton
class NetworkContactsRepository
    @Inject
    constructor(
        private val api: TranquiApi,
        private val json: Json,
    ) : ContactsRepository {
        override suspend fun contacts(): List<Contact> = apiCall(json) { api.listContacts() }.contacts

        override suspend fun create(
            name: String,
            phone: String,
        ): Contact = apiCall(json) { api.createContact(ContactCreate(name = name, phone = phone)) }
    }
