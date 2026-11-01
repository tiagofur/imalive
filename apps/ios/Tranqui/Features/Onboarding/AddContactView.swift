import SwiftUI

/// Paso 4, the memorable moment (brief): with the real names being typed,
/// the bubble shows the EXACT message the trusted person will receive —
/// mirror, not promise. The invitation itself is the backend's job (the
/// POST sends the real welcome); WhatsApp is only the offline fallback.
/// The honest skip and the free-plan quota copy live here too.
struct AddContactView: View {
    let model: OnboardingModel

    var body: some View {
        if let saved = model.state.savedContactName {
            savedView(savedName: saved)
        } else {
            contactForm
        }
    }

    private func savedView(savedName: String) -> some View {
        Form {
            Section {
                Text(String(format: String(localized: "onboarding.contact_saved"), savedName))
                    .font(.title3.bold())
                    .listRowBackground(Color.clear)
                    .listRowInsets(EdgeInsets())
            }
            if model.state.quotaNotice {
                Section {
                    Text(String(localized: "onboarding.contact_limit"))
                        .font(.footnote)
                        .foregroundStyle(.secondary)
                }
            }
            Section {
                HeroButton(
                    title: String(localized: "onboarding.contact_invite_another"),
                    action: model.inviteAnotherTapped
                )
                .listRowBackground(Color.clear)
                .listRowInsets(EdgeInsets())
                HeroButton(
                    title: String(localized: "onboarding.contact_finish"),
                    action: model.skipContactTapped
                )
                .listRowBackground(Color.clear)
                .listRowInsets(EdgeInsets())
            }
        }
    }

    private var contactForm: some View {
        Form {
            Section {
                Text(String(localized: "onboarding.contact_title"))
                    .font(.title2.bold())
                    .listRowBackground(Color.clear)
                    .listRowInsets(EdgeInsets())
                Text(String(localized: "onboarding.contact_sub"))
                    .font(.subheadline)
                    .foregroundStyle(.secondary)
                    .listRowBackground(Color.clear)
                    .listRowInsets(EdgeInsets())
            }
            Section {
                MessagePreviewCard(
                    inviterName: model.state.name,
                    contactName: model.state.contactName
                )
                .listRowBackground(Color.clear)
                .listRowInsets(EdgeInsets())
            }

            Section {
                TextField(
                    String(localized: "onboarding.contact_name_label"),
                    text: Binding(
                        get: { model.state.contactName },
                        set: { model.contactNameChanged($0) }
                    )
                )
                .textContentType(.name)
                TextField(
                    String(localized: "onboarding.contact_phone_label"),
                    text: Binding(
                        get: { model.state.contactPhone },
                        set: { model.contactPhoneChanged($0) }
                    ),
                    prompt: Text(String(localized: "onboarding.contact_phone_hint"))
                )
                .keyboardType(.phonePad)
                .textContentType(.telephoneNumber)
                .accessibilityLabel(String(localized: "onboarding.contact_phone_label"))
                if let error = model.state.contactError {
                    Text(errorText(error))
                        .font(.footnote)
                        .foregroundStyle(.red)
                }
            }

            Section {
                HeroButton(
                    title: model.state.busy == .savingContact
                        ? String(localized: "onboarding.busy_saving_contact")
                        : String(localized: "onboarding.contact_cta"),
                    action: model.inviteTapped
                )
                .disabled(model.state.busy != nil)
                .listRowBackground(Color.clear)
                .listRowInsets(EdgeInsets())
                Button(String(localized: "onboarding.contact_skip")) {
                    model.skipContactTapped()
                }
                .frame(maxWidth: .infinity)
                .listRowBackground(Color.clear)
                .listRowInsets(EdgeInsets())
            }
        }
        .scrollDismissesKeyboard(.interactively)
    }

    private func errorText(_ error: ContactError) -> String {
        switch error {
        case .invalidPhone:
            String(localized: "onboarding.error_phone_invalid")
        case .offline:
            String(localized: "onboarding.error_offline")
        case .retry:
            String(localized: "onboarding.error_retry")
        }
    }
}
