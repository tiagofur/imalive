import Foundation

/// The profile facts the home renders (contract v0.5.0: GET /v1/me carries
/// streakDays; the pause state marks the streak as frozen, R-12).
struct Profile: Equatable, Sendable {
    let streakDays: Int
    let streakFrozen: Bool
}

/// The seam between the home and the profile API (GET /v1/me). The real
/// URLSession implementation lands with the auth flow (issue #75, onboarding):
/// until there is a session, the stub keeps the UI honest and the model
/// tested.
protocol ProfileRepository: Sendable {
    func profile() async throws -> Profile
}

/// F0 stub: static values, the same shape the network client will deliver.
struct StubProfileRepository: ProfileRepository {
    func profile() async throws -> Profile {
        Profile(streakDays: 12, streakFrozen: false)
    }
}
