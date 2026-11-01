import Foundation

/// Reads the app bundle's Localizable.strings for one language. The test
/// target is hosted in the app, so Bundle.main is the app bundle; the
/// compiled .strings files are binary property lists.
struct Lproj {
    /// The languages of the bundle (es is the development language).
    static let all = ["es", "en", "pt"]

    let localization: String

    /// The localized format string of `key` (empty when missing).
    func string(_ key: String) -> String {
        guard let path = Bundle.main.path(forResource: localization, ofType: "lproj"),
              let bundle = Bundle(path: path)
        else {
            return ""
        }
        return bundle.localizedString(forKey: key, value: nil, table: nil)
    }

    /// Every key/value pair of the localization's Localizable.strings. The
    /// guard tests scan them all so a regression cannot hide in a
    /// translation either.
    func allStrings() throws -> [String: String] {
        guard let path = Bundle.main.path(
            forResource: "Localizable",
            ofType: "strings",
            inDirectory: nil,
            forLocalization: localization
        ), let data = NSData(contentsOfFile: path) as Data? else {
            return [:]
        }
        let plist = try PropertyListSerialization.propertyList(from: data, options: [], format: nil)
        return plist as? [String: String] ?? [:]
    }
}
