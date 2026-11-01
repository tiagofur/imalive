import Foundation

#if DEBUG
    /// An auth seam for Xcode Previews only: answers nothing, records nothing,
    /// touches no session. The test fakes live in the test target.
    struct PreviewAuthRepository: AuthRepository {
        func requestOtp(phone _: String) async throws {}

        func verifyOtp(phone _: String, code _: String) async throws {}
    }
#endif
