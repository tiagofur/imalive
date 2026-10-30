import XCTest

/// Bootstrap smoke gate (issue #4): the app builds, launches in the simulator
/// and shows its root. The home PR of the chain replaces this file with the
/// "Estoy OK" assertion required by the issue.
final class LaunchSmokeTests: XCTestCase {
    @MainActor
    func testAppLaunchesAndShowsTheRootTitle() {
        let app = XCUIApplication()
        app.launch()

        XCTAssertTrue(app.staticTexts["Tranqui"].waitForExistence(timeout: 5))
    }
}
