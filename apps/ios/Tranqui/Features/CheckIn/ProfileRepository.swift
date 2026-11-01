import Foundation

/// The profile facts the home renders (contract v0.5.0: GET /v1/me carries
/// streakDays; the active pause R-11 is what freezes it, R-12).
struct Profile: Equatable, Sendable {
    let streakDays: Int
    /// Daily check-in time (HH:MM, 24h) in the user timezone (contract
    /// pattern); the home builds its next-checkin label from it.
    let checkinTime: String
    /// The pause window, when there is one (R-11); nil = no pause. The
    /// freeze is derived by the caller: only a pause still live freezes.
    let pauseUntil: Date?
}

/// The seam between the home and the profile API (GET/PATCH /v1/me; issue
/// #76 slice 2, the iOS mirror of Android's). Failures surface typed
/// (ApiError with the business code / URLError): the repo never swallows
/// them — mapping them to UI states is the model's job.
protocol ProfileRepository: Sendable {
    func profile() async throws -> Profile

    /// Partial update (PATCH /v1/me; the ritual step of the onboarding,
    /// slice 4): name + check-in time + the device timezone. The #88 fix
    /// guarantees streakDays on every 200, so the answer decodes complete;
    /// the caller consumes nothing from the body.
    func update(_ patch: ProfilePatch) async throws
}

/// Real read/update of /v1/me (issue #76 slices 2 and 4; replaces the F0
/// stub).
struct NetworkProfileRepository: ProfileRepository {
    let api: TranquiAPI

    func profile() async throws -> Profile {
        let dto = try await api.getProfile()
        return Profile(
            streakDays: dto.streakDays,
            checkinTime: dto.checkinTime,
            pauseUntil: dto.pauseUntil
        )
    }

    func update(_ patch: ProfilePatch) async throws {
        _ = try await api.updateProfile(patch) // 200; the body is not consumed
    }
}
