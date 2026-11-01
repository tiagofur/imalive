import Foundation

/// The ritual and the trusted-contact moment (issue #76 slice 4): the
/// machine's paso 3 and paso 4 intents, as an extension of the step
/// machine. Same module by design: this file is the machine too — the
/// views never touch these seams.
extension OnboardingModel {
    // MARK: - The ritual (paso 3)

    func ritualNameChanged(_ raw: String) {
        // The server accepts 1..80 characters; the field stops there
        // instead of failing later.
        state.name = String(raw.prefix(Self.maxNameLength))
        state.ritualError = nil
    }

    func ritualTimeChanged(minutesFromMidnight: Int) {
        state.checkinMinutes = max(0, min(24 * 60 - 1, minutesFromMidnight))
    }

    func ritualContinueTapped() {
        guard state.busy == nil else { return }
        guard let name = state.name.trimmingCharacters(in: .whitespacesAndNewlines)
            .nonEmpty
        else {
            state.ritualError = .invalidName
            return
        }
        let patch = ProfilePatch(
            name: name,
            checkinTime: Self.hhmm(state.checkinMinutes),
            timezone: TimeZone.current.identifier
        )
        state.busy = .savingRitual
        state.ritualError = nil
        Task {
            do {
                try await profiles.update(patch)
                state.busy = nil
                state.step = .contact
            } catch is URLError {
                state.busy = nil
                state.ritualError = .offline
            } catch {
                state.busy = nil
                state.ritualError = .retry
            }
        }
    }

    // MARK: - The contact (paso 4, the memorable moment)

    func contactNameChanged(_ raw: String) {
        state.contactName = String(raw.prefix(Self.maxNameLength))
        state.contactError = nil
        state.quotaNotice = false
    }

    func contactPhoneChanged(_ raw: String) {
        state.contactPhone = raw.filter { Self.digits.contains($0) || $0 == "+" }
        state.contactError = nil
        state.quotaNotice = false
    }

    func inviteTapped() {
        guard state.busy == nil else { return }
        guard (try? Self.e164.wholeMatch(in: state.contactPhone)) != nil else {
            state.contactError = .invalidPhone
            return
        }
        let name = state.contactName
        let phone = state.contactPhone
        state.busy = .savingContact
        state.contactError = nil
        Task {
            do {
                let saved = try await contacts.create(name: name, phone: phone)
                // The backend sent the real welcome; the mirror already
                // showed it. The flow is finished: home opens.
                await sessions.completeOnboarding()
                state.busy = nil
                state.savedContactName = saved.name
            } catch let error as ApiError {
                handleContact(error, fallback: {
                    state.busy = nil
                    state.contactError = .retry
                })
            } catch is URLError {
                state.busy = nil
                state.contactError = .offline
            } catch {
                state.busy = nil
                state.contactError = .retry
            }
        }
    }

    /// The honest skip (brief paso 4): the account works, the circle waits.
    func skipContactTapped() {
        guard state.busy == nil else { return }
        sessions.completeOnboarding()
    }

    /// "Invitar a otra persona" — same screen, clean fields. The free plan's
    /// next create answers CONTACT_LIMIT_REACHED, which is the quota copy,
    /// never an error state (brief paso 4).
    func inviteAnotherTapped() {
        state.savedContactName = nil
        state.contactName = ""
        state.contactPhone = ""
        state.contactError = nil
        state.quotaNotice = false
    }

    private func handleContact(_ error: ApiError, fallback: () -> Void) {
        switch error.code {
        case .phoneInvalid:
            state.busy = nil
            state.contactError = .invalidPhone
        case .contactLimitReached:
            // Not an error: the invitation may not have gone out, but the
            // onboarding is complete either way.
            state.busy = nil
            state.quotaNotice = true
            sessions.completeOnboarding()
        case .unauthorized, .validationFailed, .contactNotFound, .otpInvalid, .otpExpired, .rateLimited, .sosPaused, .unknown:
            fallback()
        }
    }
}
