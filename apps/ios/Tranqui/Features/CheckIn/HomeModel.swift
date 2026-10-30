import Foundation
import Observation

/// Home state for F0 + the streak the user is protecting (R-12). No
/// persistence, no network yet (docs/diseno/ux-movil.md).
@MainActor
@Observable
final class HomeModel {
    enum CheckInState: Equatable {
        case pending(nextCheckin: String)
    }

    private(set) var state: CheckInState = .pending(nextCheckin: "hoy 21:00")

    /// The streak (R-12): a failed read keeps the defaults — the home is the
    /// button, the streak is decoration and must never break it.
    private(set) var streakDays = 0
    private(set) var streakFrozen = false

    private let profiles: ProfileRepository

    /// Static until the backend client lands with the MVP (R-04: the tap will
    /// hit the contract's POST /v1/checkins, issue #2).
    var nextCheckinLabel: String {
        guard case let .pending(nextCheckin) = state else { return "" }
        return nextCheckin
    }

    init(profiles: ProfileRepository = StubProfileRepository()) {
        self.profiles = profiles
        Task { await loadStreak() }
    }

    /// Intentional seam: wired to POST /v1/checkins in the MVP.
    func checkIn() {
        // F0 has no network client yet, so the tap stays a visual seam.
    }

    /// Loads the streak from the repository. The init fires it as a Task; the
    /// tests await it directly so the assertions are deterministic.
    func loadStreak() async {
        do {
            let profile = try await profiles.profile()
            streakDays = profile.streakDays
            streakFrozen = profile.streakFrozen
        } catch {
            // Defaults: no streak shown, home intact.
        }
    }
}
