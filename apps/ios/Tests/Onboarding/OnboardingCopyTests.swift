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

    // MARK: - paso 2 (teléfono → código)

    @Test(arguments: Lproj.all)
    func thePhoneQuestionIsTheBriefs(language: String) {
        let expected = switch language {
        case "es": "¿A qué teléfono te enviamos el código?"
        case "en": "Which phone should we send the code to?"
        default: "Para qual telefone enviamos o código?"
        }
        #expect(Lproj(localization: language).string("onboarding.phone_title") == expected)
    }

    @Test(arguments: Lproj.all)
    func theCodeSubtitleNamesThePhone(language: String) {
        let expected = switch language {
        case "es": "Te lo enviamos al +52 55 1234 5678."
        case "en": "We sent it to +52 55 1234 5678."
        default: "Enviamos para +52 55 1234 5678."
        }
        let format = Lproj(localization: language).string("onboarding.code_sub")

        #expect(String(format: format, "+52 55 1234 5678") == expected)
    }

    @Test(arguments: Lproj.all)
    func otpInvalidSpeaksTheBriefsHumanCopy(language: String) {
        let expected = switch language {
        case "es": "Ese código no es. Revísalo."
        case "en": "That's not the code. Check it."
        default: "Esse código não é. Confira."
        }
        #expect(Lproj(localization: language).string("onboarding.error_otp_invalid") == expected)
    }

    @Test(arguments: Lproj.all)
    func otpExpiredSpeaksTheAutoResendCopy(language: String) {
        let expected = switch language {
        case "es": "El código caducó. Te enviamos uno nuevo."
        case "en": "The code expired. We sent you a new one."
        default: "O código venceu. Enviamos um novo."
        }
        #expect(Lproj(localization: language).string("onboarding.error_otp_expired") == expected)
    }

    @Test(arguments: Lproj.all)
    func theRateLimitCopyShowsTheLiveSeconds(language: String) {
        let expected = switch language {
        case "es": "Demasiados intentos. Prueba otra vez en 47 s."
        case "en": "Too many attempts. Try again in 47 s."
        default: "Muitas tentativas. Tente de novo em 47 s."
        }
        let format = Lproj(localization: language).string("onboarding.error_rate_limited")

        #expect(String(format: format, 47) == expected)
    }
}
