import Foundation

/// The trusted circle (R-03): adding a contact makes the backend send the
/// WhatsApp/SMS welcome — the contact never registers or installs anything.
/// The wire DTO doubles as the domain type: pure data (name + phone + ids),
/// no product rules (the 1..5 quota is enforced server-side and arrives as
/// CONTACT_LIMIT_REACHED). Failures surface typed, never swallowed.
protocol ContactsRepository: Sendable {
    func contacts() async throws -> [Contact]

    func create(name: String, phone: String) async throws -> Contact
}

struct NetworkContactsRepository: ContactsRepository {
    let api: TranquiAPI

    func contacts() async throws -> [Contact] {
        try await api.listContacts().contacts
    }

    func create(name: String, phone: String) async throws -> Contact {
        try await api.createContact(ContactCreate(name: name, phone: phone))
    }
}
