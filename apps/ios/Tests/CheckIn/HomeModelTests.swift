import Foundation
import Testing
@testable import Tranqui

@MainActor
struct HomeModelTests {
    @Test func stateStartsPendingAtNineToday() {
        let model = HomeModel()

        #expect(model.state == .pending(nextCheckin: "hoy 21:00"))
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
