import Foundation
import Testing
@testable import Tranqui

@MainActor
struct HomeModelTests {
    /// A repo the tests control: fixed values or an explosion.
    private struct FixedRepo: ProfileRepository {
        let profile: Profile

        func profile() async throws -> Profile {
            profile
        }
    }

    private struct ExplodingRepo: ProfileRepository {
        func profile() async throws -> Profile {
            throw URLError(.notConnectedToInternet)
        }
    }

    @Test func stateStartsPendingAtNineToday() {
        let model = HomeModel()

        #expect(model.state == .pending(nextCheckin: "hoy 21:00"))
        #expect(model.nextCheckinLabel == "hoy 21:00")
    }

    @Test func loadsTheStreakFromTheRepository() async {
        let model = HomeModel(profiles: FixedRepo(profile: Profile(streakDays: 12, streakFrozen: false)))
        await model.loadStreak()

        #expect(model.streakDays == 12)
        #expect(model.streakFrozen == false)
    }

    @Test func marksTheStreakFrozenWhileThePauseIsLive() async {
        let model = HomeModel(profiles: FixedRepo(profile: Profile(streakDays: 7, streakFrozen: true)))
        await model.loadStreak()

        #expect(model.streakDays == 7)
        #expect(model.streakFrozen == true)
    }

    @Test func aFailedProfileReadKeepsTheHomeIntact() async {
        let model = HomeModel(profiles: ExplodingRepo())
        await model.loadStreak()

        #expect(model.streakDays == 0)
        #expect(model.streakFrozen == false)
        #expect(model.nextCheckinLabel == "hoy 21:00")
    }

    @Test(
        arguments: [
            ("es", "Estoy OK"),
            ("en", "I'm OK"),
            ("pt", "Estou bem")
        ]
    )
    func callToActionIsLocalized(language: String, expected: String) {
        #expect(localized("checkin.cta", in: language) == expected)
    }

    @Test(
        arguments: [
            ("es", "Siguiente check-in: hoy 21:00"),
            ("en", "Next check-in: hoy 21:00"),
            ("pt", "Próximo check-in: hoy 21:00")
        ]
    )
    func nextCheckinLineIsLocalized(language: String, expected: String) {
        let format = localized("home.next_checkin", in: language)

        #expect(String(format: format, "hoy 21:00") == expected)
    }

    @Test(
        arguments: [
            ("es", "12 días seguidos 🔥"),
            ("en", "12 days in a row 🔥"),
            ("pt", "12 dias seguidos 🔥")
        ]
    )
    func streakLineIsLocalized(language: String, expected: String) {
        let format = localized("home.streak_days", in: language)

        #expect(String(format: format, 12) == expected)
    }

    @Test(
        arguments: [
            ("es", "Pausa activa · racha congelada"),
            ("en", "Pause on · streak frozen"),
            ("pt", "Pausa ativa · ofensiva congelada")
        ]
    )
    func frozenLineIsLocalized(language: String, expected: String) {
        #expect(localized("home.streak_frozen", in: language) == expected)
    }

    /// Reads a key from the app bundle's Localizable.strings for one language.
    /// The test target is hosted in the app, so Bundle.main is the app bundle.
    private func localized(_ key: String, in language: String) -> String {
        guard let path = Bundle.main.path(forResource: language, ofType: "lproj"),
              let languageBundle = Bundle(path: path)
        else {
            return ""
        }
        return languageBundle.localizedString(forKey: key, value: nil, table: nil)
    }
}
