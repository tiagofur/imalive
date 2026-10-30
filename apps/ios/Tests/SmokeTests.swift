import Testing
@testable import Tranqui

/// Placeholder gate until the first feature model lands with the home PR of
/// the chain (issue #4): proves the test target links against the app module
/// under Swift 6 strict concurrency.
@Test func appModuleLinks() {
    #expect(String(describing: TranquiApp.self) == "TranquiApp")
}
