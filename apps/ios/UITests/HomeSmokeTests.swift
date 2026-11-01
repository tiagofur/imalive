import XCTest

/// The session gate (issue #76 slice 2) at the UI level: without a session
/// the app opens on the onboarding entry point (the brief's paso-1 copy);
/// with the smoke session it opens on the home with the OK button. Launch
/// arguments force the development language (es) and the ephemeral session
/// mode, so the assertions depend on neither the simulator's language nor
/// its Keychain state.
final class HomeSmokeTests: XCTestCase {
    @MainActor
    func testWithoutASessionTheGateShowsTheOnboardingEntryPoint() {
        let app = XCUIApplication()
        app.launchArguments = smokeArguments(signedIn: false)
        app.launch()

        let promise = "Un toque al día para que los que te quieren sepan que estás bien."
        XCTAssertTrue(app.staticTexts[promise].waitForExistence(timeout: 5))
        XCTAssertTrue(app.buttons["Empezar"].exists)
    }

    @MainActor
    func testHomeShowsTheOkButtonAndTheNextCheckinStatus() {
        let app = XCUIApplication()
        app.launchArguments = smokeArguments(signedIn: true)
        app.launch()

        XCTAssertTrue(app.buttons["Estoy OK"].waitForExistence(timeout: 5))
        XCTAssertTrue(app.staticTexts["Siguiente check-in: hoy 21:00"].exists)
    }

    /// `-tranqui-session-stub` boots the gate over an ephemeral session store
    /// (seeded or empty) without touching the Keychain: both tests are
    /// order-independent and hermetic.
    private func smokeArguments(signedIn: Bool) -> [String] {
        let sessionMode = signedIn ? "signed-in" : "signed-out"
        return ["-AppleLanguages", "(es)", "-AppleLocale", "es_ES", "-tranqui-session-stub", sessionMode]
    }
}
