import Foundation

// DTOs of the week-1 surface, named 1:1 with the contract's schemas
// (openapi.yaml v0.5.0, ADR-0003). Plain Codable: the contract's keys are
// already lowerCamelCase and unknown extra keys are ignored by default; the
// #88 fix guarantees the PATCH /me 200 carries every required UserProfile
// field, so the profile decodes complete without defenses.

/// Request an OTP code for the phone (schemas/OtpRequest). Phone is E.164.
struct OtpRequest: Encodable, Sendable {
    let phone: String
}

/// Verify the OTP code received by WhatsApp/SMS (schemas/OtpVerify).
struct OtpVerify: Encodable, Sendable {
    let phone: String
    let code: String
}

/// Session token pair; no PII — the profile comes from GET /me
/// (schemas/Tokens).
struct Tokens: Decodable, Sendable {
    let accessToken: String
    let refreshToken: String
    let expiresInSeconds: Int
}

/// Rotate the session with a valid refresh token (schemas/RefreshRequest).
struct RefreshRequest: Encodable, Sendable {
    let refreshToken: String
}

/// The authenticated user's profile (schemas/UserProfile). Minimum data: no
/// email, address, notes or avatar. `pauseUntil` in the past reads as no
/// pause — the freeze is evaluated by the caller (R-11/R-12).
struct UserProfile: Decodable, Sendable {
    let id: String
    let name: String
    let phone: String
    /// Daily check-in time (HH:MM, 24h) in the user timezone.
    let checkinTime: String
    let timezone: String
    let escalationWindowMinutes: Int
    let streakDays: Int
    let pauseUntil: Date?

    private enum CodingKeys: String, CodingKey {
        case id, name, phone, checkinTime, timezone, escalationWindowMinutes, streakDays, pauseUntil
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        id = try container.decode(String.self, forKey: .id)
        name = try container.decode(String.self, forKey: .name)
        phone = try container.decode(String.self, forKey: .phone)
        checkinTime = try container.decode(String.self, forKey: .checkinTime)
        timezone = try container.decode(String.self, forKey: .timezone)
        escalationWindowMinutes = try container.decode(Int.self, forKey: .escalationWindowMinutes)
        streakDays = try container.decode(Int.self, forKey: .streakDays)
        // A degraded pauseUntil (a timestamp the wire decoder cannot parse)
        // reads as no pause instead of sinking the whole profile: the streak
        // survives, only the freeze is lost (mirror of Android's
        // parseInstant in its NetworkProfileRepository).
        pauseUntil = try? container.decodeIfPresent(Date.self, forKey: .pauseUntil)
    }
}

/// Partial profile update; omitted (nil) fields keep their values
/// (schemas/ProfilePatch). JSONEncoder skips nils, so only what the user
/// touched goes on the wire.
struct ProfilePatch: Encodable, Sendable {
    var name: String?
    var checkinTime: String?
    var timezone: String?
    var escalationWindowMinutes: Int?
}

/// New contact payload (schemas/ContactCreate). The contact never
/// registers: the POST triggers the WhatsApp/SMS welcome.
struct ContactCreate: Encodable, Sendable {
    let name: String
    let phone: String
}

/// A trusted-circle contact (schemas/Contact): name + phone, nothing else.
struct Contact: Decodable, Sendable {
    let id: String
    let name: String
    let phone: String
    let createdAt: Date
}

/// The trusted circle; product-bounded to 5, no paging (schemas/ContactList).
struct ContactList: Decodable, Sendable {
    let contacts: [Contact]
}

/// Today's recorded "I'm OK" signal (schemas/Checkin). `date` stays a
/// String: it is a cycle date (YYYY-MM-DD), not an instant.
struct Checkin: Decodable, Sendable {
    let date: String
    let checkedInAt: Date
    let replayed: Bool
}
