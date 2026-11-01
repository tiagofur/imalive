import Foundation

#if DEBUG
    /// An auth seam for Xcode Previews only: answers nothing, records nothing,
    /// touches no session. The test fakes live in the test target.
    struct PreviewAuthRepository: AuthRepository {
        func requestOtp(phone _: String) async throws {}

        func verifyOtp(phone _: String, code _: String) async throws {}
    }

    /// Preview seams for the ritual's deps (slice 4): answer nothing, touch
    /// nothing. The test fakes live in the test target.
    enum PreviewSupport {
        static var profiles: ProfileRepository {
            PreviewProfileRepository()
        }

        static var contacts: ContactsRepository {
            PreviewContactsRepository()
        }

        @MainActor
        static var sessions: SessionStore {
            SessionStore(persisting: EphemeralSessionPersisting())
        }

        struct PreviewProfileRepository: ProfileRepository {
            func profile() async throws -> Profile {
                Profile(streakDays: 0, checkinTime: "21:00", pauseUntil: nil)
            }

            func update(_: ProfilePatch) async throws {}
        }

        struct PreviewContactsRepository: ContactsRepository {
            func contacts() async throws -> [Contact] {
                []
            }

            func create(name _: String, phone _: String) async throws -> Contact {
                Contact(id: UUID().uuidString, name: "", phone: "", createdAt: Date())
            }
        }
    }
#endif
