import Foundation
import Observation

/// Home state for F0: a single pending check-in with static copy. No
/// persistence, no network (docs/diseno/ux-movil.md, home placeholder).
@MainActor
@Observable
final class HomeModel {
    enum CheckInState: Equatable {
        case pending(nextCheckin: String)
    }

    private(set) var state: CheckInState = .pending(nextCheckin: "hoy 21:00")

    /// Static until the backend client lands with the MVP (R-04: the tap will
    /// hit the contract's POST /v1/checkins, issue #2).
    var nextCheckinLabel: String {
        guard case let .pending(nextCheckin) = state else { return "" }
        return nextCheckin
    }

    /// Intentional seam: wired to POST /v1/checkins in the MVP.
    func checkIn() {
        // F0 has no network client yet, so the tap stays a visual seam.
    }
}
