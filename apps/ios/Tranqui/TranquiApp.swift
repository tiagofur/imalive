import SwiftUI

@main
struct TranquiApp: App {
    @State private var sessions: SessionStore
    @State private var home: HomeModel

    init() {
        // One stack for the whole app: the Keychain-backed session store
        // feeds the API's bearer attach and refresh; the home reads the
        // profile through the same API (issue #76 slice 2). The gate wiring
        // lands with the onboarding entry point.
        let sessions = SessionStore(persisting: KeychainSessionStore())
        let api = TranquiAPI(sessions: sessions)
        _sessions = State(initialValue: sessions)
        _home = State(initialValue: HomeModel(profiles: NetworkProfileRepository(api: api)))
    }

    var body: some Scene {
        WindowGroup {
            ContentView()
                .environment(home)
        }
    }
}
