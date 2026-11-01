import Foundation

/// The single-flight refresh (plan #76, the iOS mirror of Android's
/// TokenAuthenticator): concurrent 401s funnel through exactly ONE
/// POST /auth/refresh, and everyone in the burst retries with the rotated
/// pair. Outcome per attempt:
///
/// - `.rotated` → the caller retries the original request with the fresh pair.
/// - `.rejected` → the refresh answered 401/problem (or undecodable): the
///   session was wiped through the store, so the reactive gate walks the
///   user back to onboarding.
/// - `.unavailable` → the refresh could not reach the network (URLError):
///   the session is KEPT — an outage must never log the user out — and only
///   the original request is given up.
///
/// The actor serializes entry; the in-flight task makes the burst share one
/// rotation, and the stale-token check covers the arrival that lands after a
/// rotation already completed.
actor TokenRefresher {
    enum Outcome: Sendable {
        case rotated(StoredSession)
        case rejected
        case unavailable
    }

    private let sessions: any SessionStoring
    private var inFlight: Task<Outcome, Never>?

    init(sessions: any SessionStoring) {
        self.sessions = sessions
    }

    /// Rotates the session after a 401. `refresh` is the raw exchange the
    /// API lends for this call (no bearer, no nested refresh — the recursion
    /// guard lives at the call site).
    func refreshedSession(
        staleAccessToken: String,
        refresh: @escaping @Sendable (String) async throws -> Tokens
    ) async -> Outcome {
        // Someone in the burst is already rotating: share their flight.
        if let inFlight {
            return await inFlight.value
        }
        // A rotation already completed between the 401 and this entry: the
        // cached pair is fresh for this caller — no second round trip.
        if let current = await sessions.cachedSession(), current.accessToken != staleAccessToken {
            return .rotated(current)
        }
        let task = Task { [sessions] in
            guard let current = await sessions.cachedSession() else {
                return Outcome.rejected // nothing left to protect
            }
            do {
                let tokens = try await refresh(current.refreshToken)
                let rotated = StoredSession(
                    accessToken: tokens.accessToken,
                    refreshToken: tokens.refreshToken,
                    expiresAt: Date().addingTimeInterval(TimeInterval(tokens.expiresInSeconds))
                )
                await sessions.save(rotated)
                return Outcome.rotated(rotated)
            } catch is URLError {
                // Transient outage: keep the session, give up this request only.
                return Outcome.unavailable
            } catch {
                // 401/problem/undecodable answer: the session is invalid.
                await sessions.clear()
                return Outcome.rejected
            }
        }
        inFlight = task
        let outcome = await task.value
        inFlight = nil
        return outcome
    }
}
