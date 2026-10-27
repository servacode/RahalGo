package com.rahalgo.shared.merchant

/**
 * ══════════════════════════════════════════════════════════════════════
 * **موظّفو المتجر ورقمُ الطلبات حسب الدوام** (قرارُ المالك ٢٠٢٦-١٠-٢٠)
 * ══════════════════════════════════════════════════════════════════════
 *
 * كما يكتبها المحرّك: `server/order_phones_handlers.go` و`identity/staff.go`.
 */

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

/**
 * **رقمُ الطلبات** — `full` رقمٌ واحد (رقمُ المتجر كما كان)، أو `split` صباحيٌّ
 * ومسائيٌّ بساعةِ تبديل «HH:MM» بتوقيت سوريا.
 */
@Serializable
data class OrderPhones(
    val mode: String = "full",
    @SerialName("morning_phone") val morning: String = "",
    @SerialName("evening_phone") val evening: String = "",
    @SerialName("evening_from") val eveningFrom: String = "16:00",
)

/** **موظّفٌ كما يُعرض** — بلا كلمةِ سرّه. */
@Serializable
data class StaffMember(
    val id: String = "",
    val phone: String = "",
    val name: String = "",
    @SerialName("created_at") val createdAt: String? = null,
    @SerialName("last_login_at") val lastLoginAt: String? = null,
)

@Serializable
data class StaffPage(val staff: List<StaffMember> = emptyList())

@Serializable
data class StaffInput(val phone: String, val name: String)

/** **الموظّفُ وكلمةُ سرّه** — تُعرض مرّةً لمن أضافه، و`sent` إن وصلته واتساب. */
@Serializable
data class StaffAdded(
    val staff: StaffMember = StaffMember(),
    val password: String = "",
    val sent: Boolean = false,
)

@Serializable
data class StaffRemoved(val removed: Boolean = false)
