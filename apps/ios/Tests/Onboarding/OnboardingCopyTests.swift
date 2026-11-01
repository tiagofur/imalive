import Foundation
import Testing
@testable import Tranqui

/// Pins the brief's welcome copy per language (the screens' words are the
/// product): table-driven like the home localization tests. The guard test
/// polices what may NOT be said; these pin what MUST be.
///
/// The expectations are switch expressions instead of array literals: the
/// table rows carry the accented product copy and both formatters disagree
/// on trailing commas in multi-line lists (swiftformat adds them,
/// swiftlint's trailing_comma forbids them), so the file carries no
/// collection literal at all.
struct OnboardingCopyTests {
    @Test(arguments: Lproj.all)
    func theWelcomeCallToActionIsLocalized(language: String) {
        let expected = switch language {
        case "es": "Empezar"
        case "en": "Get started"
        default: "Começar"
        }
        #expect(Lproj(localization: language).string("onboarding.welcome_cta") == expected)
    }

    @Test(arguments: Lproj.all)
    func theProgressSpeaksInRitualNotBureaucracy(language: String) {
        let expected = switch language {
        case "es": "Paso 2 de 4"
        case "en": "Step 2 of 4"
        default: "Passo 2 de 4"
        }
        let format = Lproj(localization: language).string("onboarding.progress_a11y")

        #expect(String(format: format, 2) == expected)
    }

    @Test(arguments: Lproj.all)
    func theWelcomeTitleIsTheBriefPromise(language: String) {
        let expected = switch language {
        case "es": "Un toque al día para que los que te quieren sepan que estás bien."
        case "en": "One tap a day so the people who love you know you're okay."
        default: "Um toque por dia para que quem te ama saiba que você está bem."
        }
        #expect(Lproj(localization: language).string("onboarding.welcome_title") == expected)
    }

    @Test(arguments: Lproj.all)
    func theWelcomeSubSpeaksTheHonestPromise(language: String) {
        let expected = switch language {
        case "es": "Si un día no confirmas, tu gente lo sabe al instante y alguien puede comprobar cómo estás."
        case "en": "If one day you don't confirm, your people know right away and someone can check on you."
        default:
            "Se um dia você não confirmar, quem te ama fica sabendo na hora " +
                "e alguém pode verificar como você está."
        }
        #expect(Lproj(localization: language).string("onboarding.welcome_sub") == expected)
    }

    @Test(arguments: Lproj.all)
    func theDisclaimerRendersWithTheProductName(language: String) {
        let expected = switch language {
        case "es": "TRANQUI avisa a tu gente; no es un servicio de emergencia y no sustituye al 911/112."
        case "en": "TRANQUI lets your people know; it is not an emergency service and does not replace 911/112."
        default: "TRANQUI avisa quem você ama; não é um serviço de emergência e não substitui o 911/112."
        }
        let format = Lproj(localization: language).string("onboarding.welcome_disclaimer")

        #expect(String(format: format, "TRANQUI") == expected)
    }
}
