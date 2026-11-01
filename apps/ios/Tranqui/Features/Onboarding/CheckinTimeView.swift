import SwiftUI

/// Paso 3, the ritual (brief): the user's name — the welcome message will
/// call them by it — and the daily reminder time. One protagonist action per
/// screen: type your name, pick your hour, continue.
struct CheckinTimeView: View {
    let model: OnboardingModel

    @State private var showTimePicker = false

    var body: some View {
        Form {
            Section {
                Text(String(localized: "onboarding.ritual_title"))
                    .font(.title2.bold())
                    .listRowBackground(Color.clear)
                    .listRowInsets(EdgeInsets())
            }
            Section {
                TextField(
                    String(localized: "onboarding.ritual_name_label"),
                    text: Binding(
                        get: { model.state.name },
                        set: { model.ritualNameChanged($0) }
                    )
                )
                .textContentType(.name)
                .accessibilityLabel(String(localized: "onboarding.ritual_name_label"))
                if let error = model.state.ritualError, error == .invalidName {
                    Text(String(localized: "onboarding.ritual_name_error"))
                        .font(.footnote)
                        .foregroundStyle(.red)
                }
            } header: {
                Text(String(localized: "onboarding.ritual_name_label"))
            }

            Section {
                Text(String(localized: "onboarding.ritual_time_sub"))
                    .font(.footnote)
                    .foregroundStyle(.secondary)
                DatePicker(
                    String(localized: "onboarding.ritual_time_label"),
                    selection: timeBinding,
                    displayedComponents: .hourAndMinute
                )
                .datePickerStyle(.wheel)
                .accessibilityHint(String(
                    format: String(localized: "onboarding.ritual_time_a11y"),
                    Self.hhmm(model.state.checkinMinutes)
                ))
            } header: {
                Text(String(localized: "onboarding.ritual_time_title"))
            }

            Section {
                HeroButton(
                    title: model.state.busy == .savingRitual
                        ? String(localized: "onboarding.busy_saving_ritual")
                        : String(localized: "onboarding.ritual_cta"),
                    action: model.ritualContinueTapped
                )
                .disabled(model.state.busy != nil)
                .listRowBackground(Color.clear)
                .listRowInsets(EdgeInsets())
            }
        }
        .scrollDismissesKeyboard(.interactively)
    }

    private var timeBinding: Binding<Date> {
        Binding(
            get: { Self.date(minutes: model.state.checkinMinutes) },
            set: { model.ritualTimeChanged(minutesFromMidnight: Self.minutes(from: $0)) }
        )
    }

    private static func date(minutes: Int) -> Date {
        Calendar.current.date(
            bySettingHour: minutes / 60, minute: minutes % 60, second: 0, of: Date()
        ) ?? Date()
    }

    private static func minutes(from date: Date) -> Int {
        let parts = Calendar.current.dateComponents([.hour, .minute], from: date)
        return (parts.hour ?? 21) * 60 + (parts.minute ?? 0)
    }

    private static func hhmm(_ minutes: Int) -> String {
        String(format: "%02d:%02d", minutes / 60, minutes % 60)
    }
}
