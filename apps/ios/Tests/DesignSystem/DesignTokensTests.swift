import SwiftUI
import Testing
@testable import Tranqui
import UIKit

/// The provisional tokens (issue #76 slice 3): the one behavior they must
/// already honor is the mandatory dark mode (design-system.md) — every
/// brand token resolves a lifted dark variant, never the light one.
struct DesignTokensTests {
    @Test(arguments: [("primary", DesignTokens.primary), ("critical", DesignTokens.critical)])
    func brandTokensLiftForDarkMode(name: String, token: Color) {
        let light = UIColor(token).resolvedColor(with: UITraitCollection(userInterfaceStyle: .light))
        let dark = UIColor(token).resolvedColor(with: UITraitCollection(userInterfaceStyle: .dark))

        #expect(light != dark, "token \(name) must resolve a distinct dark variant")
    }

    @Test func isLiquidGlassTracksTheOS() {
        // The gate mirrors the runtime: true from iOS 26 (the Liquid Glass
        // design language), false on the supported legacy OSes.
        let major = ProcessInfo.processInfo.operatingSystemVersion.majorVersion
        #expect(DesignTokens.isLiquidGlass == (major >= 26))
    }
}
