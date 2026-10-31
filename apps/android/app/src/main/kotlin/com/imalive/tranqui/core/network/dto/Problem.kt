package com.imalive.tranqui.core.network.dto

import kotlinx.serialization.Serializable

// RFC 9457 problem details (contract v0.5.0). `code` is the stable business
// code the apps switch on for copy and decisions (api-guidelines).
@Serializable
data class Problem(
    val type: String,
    val title: String,
    val status: Int,
    val code: String,
    val detail: String? = null,
    val instance: String? = null,
)
