import Foundation
import Testing
@testable import Tranqui

/// The legal guard of the onboarding (brief §5, "la promesa honesta"; the
/// iOS par of Android's OnboardingCopyGuardTest): the disclaimer's literal
/// wording — "no es un servicio de emergencia y no sustituye al 911/112" —
/// must survive every future copy edit, and "emergencia médica" or safety
/// promises are forbidden everywhere, in every language of the bundle. This
/// also protects the App Store review (producto.md §8).
struct OnboardingCopyGuardTest {
    private static let forbidden = ["emergencia médica", "te protege", "siempre a salvo"]

    @Test(arguments: Lproj.all)
    func noStringPromisesEmergencyCapabilities(localization: String) throws {
        let strings = try Lproj(localization: localization).allStrings()
        #expect(!strings.isEmpty, "the \(localization).lproj strings table must not be empty")

        for (key, value) in strings {
            for phrase in Self.forbidden {
                #expect(
                    !value.localizedLowercase.contains(phrase),
                    "\(localization).lproj \(key) contains the forbidden phrase \"\(phrase)\": \"\(value)\""
                )
            }
        }
    }

    @Test(arguments: Lproj.all)
    func theWelcomeCarriesTheHonestDisclaimer(localization: String) throws {
        let disclaimer = try #require(
            Lproj(localization: localization).allStrings()["onboarding.welcome_disclaimer"],
            "onboarding.welcome_disclaimer must exist in \(localization).lproj"
        )

        #expect(disclaimer.contains("911/112"), "the disclaimer must name the real emergency numbers")
        #expect(disclaimer.contains("%@"), "the disclaimer interpolates the product name (issue #5)")
    }
}
