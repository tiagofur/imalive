import Foundation
import Observation

/// Home state for the real profile consumption (R-12): the next-checkin
/// schedule and the streak the user is protecting. nextCheckinLabel keeps
/// the default copy until the profile read answers with the real slot
/// (GET /v1/me).
@MainActor
@Observable
final class HomeModel {
    enum CheckInState: Equatable {
        case pending(nextCheckin: String)
    }

    /// The default copy until the profile answers with the real slot.
    private static let defaultCheckinLabel = "hoy 21:00"

    private(set) var state: CheckInState = .pending(nextCheckin: HomeModel.defaultCheckinLabel)

    /// The streak (R-12): a failed read keeps the defaults — the home is the
    /// button, the streak is decoration and must never break it.
    private(set) var streakDays = 0
    private(set) var streakFrozen = false

    private let profiles: any ProfileRepository

    /// Static until the MVP check-in flow (R-04: the tap will hit the
    /// contract's POST /v1/checkins, issue #2).
    var nextCheckinLabel: String {
        guard case let .pending(nextCheckin) = state else { return "" }
        return nextCheckin
    }

    init(profiles: any ProfileRepository) {
        self.profiles = profiles
        Task { await loadStreak() }
    }

    /// Intentional seam: wired to POST /v1/checkins in the MVP.
    func checkIn() {
        // The tap stays a visual seam until the check-in flow lands.
    }

    /// Loads the streak from the repository. The init fires it as a Task; the
    /// tests await it directly so the assertions are deterministic.
    func loadStreak() async {
        do {
            let profile = try await profiles.profile()
            streakDays = profile.streakDays
            state = .pending(nextCheckin: "hoy \(profile.checkinTime)")
            // The pause (R-11) freezes the streak while it is live; an
            // expired pauseUntil reads as no pause.
            streakFrozen = profile.pauseUntil.map { $0 > Date() } ?? false
        } catch {
            // ApiError or offline: defaults, home intact.
        }
    }
}
