import XCTest

/// Criterion of issue #4 (triaged): the root view contains the "Estoy OK"
/// text, asserted inside the xcodebuild test gate. Launch arguments force the
/// development language (es) so the assertion does not depend on the
/// simulator's language.
final class HomeSmokeTests: XCTestCase {
    @MainActor
    func testHomeShowsTheOkButtonAndTheNextCheckinStatus() {
        let app = XCUIApplication()
        app.launchArguments = ["-AppleLanguages", "(es)", "-AppleLocale", "es_ES"]
        app.launch()

        XCTAssertTrue(app.buttons["Estoy OK"].waitForExistence(timeout: 5))
        XCTAssertTrue(app.staticTexts["Siguiente check-in: hoy 21:00"].exists)
    }
}
