import SwiftUI

/// R-04 (home = the button): one tap for "Estoy OK" plus the next-checkin
/// status and the streak the user is protecting (R-12), read from the real
/// profile (issue #76 slice 2).
struct HomeView: View {
    let model: HomeModel

    var body: some View {
        NavigationStack {
            VStack(spacing: 24) {
                if model.streakDays > 0 {
                    VStack(spacing: 4) {
                        Text(String(format: String(localized: "home.streak_days"), model.streakDays))
                            .font(.title2.bold())
                        if model.streakFrozen {
                            Text(String(localized: "home.streak_frozen"))
                                .font(.footnote)
                                .foregroundStyle(.secondary)
                        }
                    }
                }
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
