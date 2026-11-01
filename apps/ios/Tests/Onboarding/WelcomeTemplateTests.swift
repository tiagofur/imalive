import Foundation
import Testing
@testable import Tranqui

/// The mirror tests (issue #76 slice 4): the pure interpolation against the
/// server-rendered sentence, character for character.
struct WelcomeTemplateTests {
    /// The exact welcomeMessageV1 string, line-split for the lint.
    private let template = WelcomeTemplate(
        text: "%1$@: %2$@ te ha añadido como persona de confianza en %3$@, "
            + "la app con la que confirma a diario que está bien. "
            + "No tienes que hacer nada. Solo te escribiremos si %2$@ pasa muchas horas sin confirmar, "
            + "para que puedas comprobar cómo está. "
            + "%3$@ es un servicio de avisos entre personas; no es un servicio de emergencias.",
        inviterFallback: "[tu nombre]",
        contactFallback: "[su nombre]"
    )

    @Test func thePreviewIsTheServerSentenceCharacterForCharacter() {
        let preview = welcomePreview(
            template: template,
            inviterName: "Ana",
            contactName: "María",
            appName: "Tranqui"
        )
        #expect(preview == "Hola María: Ana te ha añadido como persona de confianza en Tranqui, "
            + "la app con la que confirma a diario que está bien. No tienes que hacer nada. "
            + "Solo te escribiremos si Ana pasa muchas horas sin confirmar, para que puedas comprobar cómo está. "
            + "Tranqui es un servicio de avisos entre personas; no es un servicio de emergencias.")
    }

    @Test func blankNamesGetTheNeutralPlaceholders() {
        let preview = welcomePreview(
            template: template,
            inviterName: "  ",
            contactName: "",
            appName: "Tranqui"
        )
        #expect(preview.hasPrefix("Hola [su nombre]: [tu nombre] te ha añadido"))
    }
}
