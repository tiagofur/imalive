import Foundation
@testable import Tranqui

/// Fakes for the ritual suites (issue #76 slice 4): recording stubs whose
/// failures the tests inject explicitly.
final class StubProfileRepo: ProfileRepository, @unchecked Sendable {
    private(set) var updates: [ProfilePatch] = []
    var updateError: Error?

    func profile() async throws -> Profile {
        Profile(streakDays: 0, checkinTime: "21:00", pauseUntil: nil)
    }

    func update(_ patch: ProfilePatch) async throws {
        if let updateError {
            throw updateError
        }
        updates.append(patch)
    }
}

final class StubContactsRepo: ContactsRepository, @unchecked Sendable {
    private(set) var created: [(name: String, phone: String)] = []
    var createError: Error?
    /// The next create answers this instead of a fresh contact (the quota).
    var limitReached = false

    func contacts() async throws -> [Contact] {
        []
    }

    func create(name: String, phone: String) async throws -> Contact {
        if let createError {
            throw createError
        }
        if limitReached {
            throw ApiError(
                code: .contactLimitReached,
                httpStatus: 422,
                retryAfterSeconds: nil,
                type: nil,
                detail: nil
            )
        }
        created.append((name, phone))
        return Contact(id: UUID().uuidString, name: name, phone: phone, createdAt: Date())
    }
}
