import Foundation
import Observation

/// Home state (R-04 + R-12): the session gate (which surface owns the
/// screen), the next-checkin schedule and the streak the user is protecting.
/// nextCheckinLabel keeps the default copy until the profile read answers
/// with the real slot (GET /v1/me).
@MainActor
@Observable
final class HomeModel {
    enum CheckInState: Equatable {
        case pending(nextCheckin: String)
    }

    /// The default copy until the profile answers with the real slot.
    private static let defaultCheckinLabel = "hoy 21:00"

    /// The gate follows the session store reactively (issue #76 slice 2).
    private(set) var gate: HomeGate = .loading
    private(set) var state: CheckInState = .pending(nextCheckin: HomeModel.defaultCheckinLabel)

    /// The streak (R-12): a failed read keeps the defaults — the home is the
    /// button, the streak is decoration and must never break it.
    private(set) var streakDays = 0
    private(set) var streakFrozen = false

    private let profiles: any ProfileRepository
    private let sessions: SessionStore

    /// Static until the MVP check-in flow (R-04: the tap will hit the
    /// contract's POST /v1/checkins, issue #2).
    var nextCheckinLabel: String {
        guard case let .pending(nextCheckin) = state else { return "" }
        return nextCheckin
    }

    /// The session store answers synchronously at init (the Keychain read is
    /// atomic), so the gate is resolved before the first frame.
    init(profiles: any ProfileRepository, sessions: SessionStore) {
        self.profiles = profiles
        self.sessions = sessions
        reactToSession()
        observeSessionGate()
    }

    /// Intentional seam: wired to POST /v1/checkins in the MVP.
    func checkIn() {
        // The tap stays a visual seam until the check-in flow lands.
    }

    /// The gate transition (Android's combine collector, in Observation
    /// terms): without a session the onboarding owns the screen; a session
    /// lands home and loads the streak; a token rotation rewrites the store
    /// while home is up — nothing reloads. Exposed for the tests: they drive
    /// the reaction directly so the assertions stay deterministic.
    func reactToSession() {
        switch sessions.gate {
        case .loading:
            break // the store has not answered yet; keep whatever we show
        case .needsOnboarding:
            guard gate != .needsOnboarding else { return }
            resetToDefaults()
            gate = .needsOnboarding
        case .ready:
            guard gate != .ready else { return }
            gate = .ready
            Task { await loadStreak() }
        }
    }

    /// Loads the streak from the repository. The ready transition fires it as
    /// a Task; the tests await it directly so the assertions are deterministic.
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

    /// A session wiped by the refresher (a rejected refresh) must not leak
    /// the previous user's data into the onboarding entry point: everything
    /// back to the defaults.
    private func resetToDefaults() {
        state = .pending(nextCheckin: HomeModel.defaultCheckinLabel)
        streakDays = 0
        streakFrozen = false
    }

    /// Continuous gate observation without Combine: Observation tracks the
    /// store's gate; on every change the reaction re-runs and re-registers
    /// (the pattern for observing @Observable outside SwiftUI views). The
    /// re-arm happens BEFORE handling so a change landing mid-reaction is
    /// never missed — the reaction always converges to the store's gate.
    private func observeSessionGate() {
        withObservationTracking { [weak self] in
            _ = self?.sessions.gate
        } onChange: { [weak self] in
            Task { @MainActor [weak self] in
                self?.observeSessionGate()
                self?.reactToSession()
            }
        }
    }
}
