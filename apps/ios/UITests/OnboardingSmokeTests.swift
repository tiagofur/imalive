import XCTest

/// The slice-3 onboarding flow at the UI level (hermetic, no network):
/// without a session the app opens on the brief's welcome — promise, hero
/// CTA and the honest disclaimer — and "Empezar" lands on the phone step.
/// The OTP wire behavior is pinned by OnboardingModelTests; the full flow
/// smoke (with the stubbed API) lands with slice 4.
final class OnboardingSmokeTests: XCTestCase {
    @MainActor
    func testEmpezarAdvancesFromTheWelcomeToThePhoneStep() {
        let app = XCUIApplication()
        app.launchArguments = smokeArguments()
        app.launch()

        // Paso 1: the promise, the ritual progress, the hero CTA and the
        // disclaimer in a low voice.
        let promise = "Un toque al día para que los que te quieren sepan que estás bien."
        XCTAssertTrue(app.staticTexts[promise].waitForExistence(timeout: 5))
        XCTAssertTrue(app.otherElements["Paso 1 de 4"].waitForExistence(timeout: 5))
        let disclaimer = "Tranqui avisa a tu gente; no es un servicio de emergencia y no sustituye al 911/112."
        XCTAssertTrue(app.staticTexts[disclaimer].exists)
        attachScreenshot(named: "onboarding-paso1-bienvenida")

        app.buttons["Empezar"].tap()

        // Paso 2, fase teléfono.
        XCTAssertTrue(app.staticTexts["¿A qué teléfono te enviamos el código?"].waitForExistence(timeout: 5))
        XCTAssertTrue(app.staticTexts["Solo para entrar. Tu número no se muestra a nadie."].exists)
        XCTAssertTrue(app.textFields["Tu teléfono"].exists)
        XCTAssertTrue(app.otherElements["Paso 2 de 4"].exists)
        attachScreenshot(named: "onboarding-paso2-telefono")
    }

    /// Visual evidence of the flow, riding in the result bundle (the
    /// simulator cannot inject taps outside XCUITest, so this is how the
    /// intermediate steps get screenshotted).
    @MainActor
    private func attachScreenshot(named name: String) {
        let attachment = XCTAttachment(screenshot: XCUIScreen.main.screenshot())
        attachment.name = name
        attachment.lifetime = .keepAlways
        add(attachment)
    }

    /// Forces the development language (es) and the ephemeral session mode,
    /// so the assertions depend on neither the simulator's language nor its
    /// Keychain state.
    private func smokeArguments() -> [String] {
        ["-AppleLanguages", "(es)", "-AppleLocale", "es_ES", "-tranqui-session-stub", "signed-out"]
    }
}
