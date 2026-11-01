package com.imalive.tranqui.feature.onboarding

import android.content.ActivityNotFoundException
import android.content.Context
import android.content.Intent

// The manual invitation fallback (plan of issue #75, slice 4): the CTA sends
// POST /v1/contacts and the backend messages the contact — the intent fires
// only from the failure states, so the trusted person never gets two
// messages. WhatsApp first (explicit package), the system share sheet with
// the same text when it is not installed (R-03: the contact registers
// nothing).
object WhatsAppInvitation {
    private const val WHATSAPP_PACKAGE = "com.whatsapp"

    fun share(
        context: Context,
        text: String,
    ) {
        try {
            context.startActivity(shareIntent(text, WHATSAPP_PACKAGE))
        } catch (_: ActivityNotFoundException) {
            context.startActivity(Intent.createChooser(shareIntent(text, null), null))
        }
    }

    private fun shareIntent(
        text: String,
        packageName: String?,
    ): Intent =
        Intent(Intent.ACTION_SEND)
            .setType("text/plain")
            .putExtra(Intent.EXTRA_TEXT, text)
            .apply { packageName?.let(::setPackage) }
}
