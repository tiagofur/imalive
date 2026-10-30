import SwiftUI

/// R-04 (home = the button): one tap for "Estoy OK" plus the next-checkin
/// status. F0 scaffold: static schedule, no navigation (docs/diseno/ux-movil.md).
struct HomeView: View {
    @State private var model = HomeModel()

    var body: some View {
        NavigationStack {
            VStack(spacing: 24) {
                Text(nextCheckinText)
                    .font(.body)
                Button(String(localized: "checkin.cta")) {
                    model.checkIn()
                }
                .buttonStyle(.borderedProminent)
            }
            .navigationTitle("Tranqui")
        }
    }

    private var nextCheckinText: String {
        String(format: String(localized: "home.next_checkin"), model.nextCheckinLabel)
    }
}
