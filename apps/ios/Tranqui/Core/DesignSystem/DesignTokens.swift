import SwiftUI
import UIKit

/// The provisional design tokens (issue #76 slice 3, the iOS par of
/// Android's Theme.kt): `/design/tokens.json` does not exist yet and
/// `brand/primary` awaits the product name (#5), so the brief's supuesto
/// applies — `ok/primary` stands in as the hero color. This file is the
/// home of named token values: its literals ARE the values (pending the
/// generator), which is why raw numbers live here and nowhere else.
///
/// The type roles ride the Apple text styles nearest the design-system
/// scale (display 32/40, headline 24/32, title 18/26, body 16/24, label
/// 13/18) so Dynamic Type scales the whole flow for free; the exact type
/// ramp arrives with tokens.json and the visual-world new-work.
enum DesignTokens {
    // ok/primary: AA on white; lifted for dark AA (dark mode is mandatory —
    // the nightly check-in is real, design-system.md).
    private static let okPrimaryLight = UIColor(red: 0x2E / 255.0, green: 0x7D / 255.0, blue: 0x5B / 255.0, alpha: 1)
    private static let okPrimaryDark = UIColor(red: 0x8A / 255.0, green: 0xD6 / 255.0, blue: 0xAE / 255.0, alpha: 1)

    // incident/critical: inline errors only; reserved, never decorative.
    private static let incidentCriticalLight = UIColor(
        red: 0xC6 / 255.0,
        green: 0x3B / 255.0,
        blue: 0x33 / 255.0,
        alpha: 1
    )
    private static let incidentCriticalDark = UIColor(
        red: 0xF2 / 255.0,
        green: 0xB8 / 255.0,
        blue: 0xB5 / 255.0,
        alpha: 1
    )

    /// brand/primary, provisional = ok/primary (the product name is #5, the
    /// final values live in the tokens.json to come).
    static let primary = Color(uiColor: UIColor { trait in
        trait.userInterfaceStyle == .dark ? DesignTokens.okPrimaryDark : DesignTokens.okPrimaryLight
    })

    /// Inline error color (incident/critical).
    static let critical = Color(uiColor: UIColor { trait in
        trait.userInterfaceStyle == .dark ? DesignTokens.incidentCriticalDark : DesignTokens.incidentCriticalLight
    })

    /// The display role (brief paso 1: "display 32/40").
    static let display = Font.largeTitle

    /// The headline role of the steps (brief: 24/32).
    static let headline = Font.title2

    /// The title role (brief: 18/26) — the hero button's label.
    static let title = Font.title3

    /// The body role (brief: 16/24).
    static let body = Font.callout

    /// The label role (brief: 13/18) — the legal voice of the disclaimers.
    static let label = Font.footnote

    /// The hero CTA height (the OkButton DNA; targets ≥ 44 pt).
    static let heroHeight: CGFloat = 56
}

extension Bundle {
    /// The product display name (CFBundleDisplayName, falling back to
    /// CFBundleName): the single place "Tranqui" lives — resolving #5 means
    /// changing one value in project.yml, no copy edits anywhere.
    var productDisplayName: String {
        let display = object(forInfoDictionaryKey: "CFBundleDisplayName") as? String
        let name = object(forInfoDictionaryKey: "CFBundleName") as? String
        return display ?? name ?? ""
    }
}
