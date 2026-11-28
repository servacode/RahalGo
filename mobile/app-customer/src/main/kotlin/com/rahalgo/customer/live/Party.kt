package com.rahalgo.customer.live

import android.content.Context
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import com.rahalgo.customer.R
import com.rahalgo.customer.orders.statusText
import com.rahalgo.design.Rahal
import com.rahalgo.shared.customer.CustomerApi
import com.rahalgo.shared.customer.PartyView
import com.rahalgo.ui.AppCore
import com.rahalgo.ui.RahalButton
import com.rahalgo.ui.RahalCard
import com.rahalgo.ui.StatusTone
import com.rahalgo.ui.Tone
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.launch

/**
 * ══════════════════════════════════════════════════════════════════════
 * **الطرفُ الآخر يشارك موقعَه من تطبيقنا** (قرارُ المالك ٢٠٢٦-١١-٢٨)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **حين تطلب المنصّةُ موقعَ هاتفٍ عليه حسابُ زبون** (أمانةٌ أو توصيلةُ متجرٍ أو طلبُ المكتب) يصله إشعارٌ
 * وبطاقةٌ «نعم/لا». **«نعم» موافقةٌ وتبدأ بعدها المشاركةُ بلا زرٍّ ثانٍ** (`LiveShareService` بنمط الطرف).
 *
 * **حالٌ واحدةٌ للتطبيق** — تقرؤها اللافتةُ والنافذةُ والاستئناف. والقراراتُ في `PartyPolicy`.
 */
object Party {

    /** **طلباتي كطرفٍ آخر** — المفتوحةُ وحدَها، كما يردّها المحرّك. */
    var orders: List<PartyView> by mutableStateOf(emptyList())
        private set

    /** **الطلبُ الذي تُعرض بطاقتُه الآن** — من لافتةٍ أو إشعار. */
    var promptId: String? by mutableStateOf(null)
        private set

    /** **جوابٌ يُرسَل الآن** — يمنع الضغطَ المزدوج. */
    var busy by mutableStateOf(false)
        private set

    /** **يقرأ طلباتي** — عند الفتح وكلَّ نبضة. **ومن أخفق بقيت القائمةُ كما كانت.** */
    fun refresh(scope: CoroutineScope) {
        val core = AppCore.get()
        if (core.session.refreshToken().isEmpty()) {
            clear()
            return
        }
        scope.launch {
            try {
                val list = CustomerApi(core.api).partyOrders()
                orders = list
                // **بطاقةٌ لطلبٍ لم يعد مفتوحاً أو أُجيب عنه تُغلق.**
                if (PartyPolicy.promptTarget(list, promptId) == null) promptId = null
            } catch (e: CancellationException) {
                throw e
            } catch (_: Exception) {
                // **الشبكةُ تسقط** — النبضةُ التالية تعيد المحاولة.
            }
        }
    }

    /** **يمحو الحالَ** — خروجٌ من الحساب أو ضيف. */
    fun clear() {
        orders = emptyList()
        promptId = null
        busy = false
    }

    /** **يفتح بطاقةَ طلبٍ بعينه** — من اللافتة أو الإشعار؛ ويجلب الطلبَ إن لم يكن في القائمة. */
    fun open(scope: CoroutineScope, orderId: String) {
        promptId = orderId
        refresh(scope)
    }

    fun dismiss() {
        promptId = null
    }

    /**
     * **جوابُ «نعم/لا»** — `onYes` يُنادى بعد قبول المحرّك لـ«نعم» لتبدأ المشاركةُ.
     * وفشلُ النداء يُعرض ولا تبدأ مشاركة.
     */
    fun answer(
        context: Context,
        scope: CoroutineScope,
        view: PartyView,
        yes: Boolean,
        onYes: (PartyView) -> Unit,
    ) {
        if (busy) return
        busy = true
        scope.launch {
            try {
                val updated = CustomerApi(AppCore.get().api).partyAnswer(view.orderId, if (yes) "yes" else "no")
                orders = orders.map { if (it.orderId == view.orderId) updated else it }
                promptId = null
                // **و«نعم» تمحو «أوقفتُها» القديمة** — من غيّر رأيه وأعاد الموافقة أراد المشاركة.
                if (yes) {
                    LiveStopped.remove(context, PartyPolicy.stopKey(view.orderId))
                    onYes(updated)
                }
            } catch (e: CancellationException) {
                throw e
            } catch (e: Exception) {
                com.rahalgo.ui.Flash.fail(com.rahalgo.ui.apiError(context, e))
            } finally {
                busy = false
            }
        }
    }

    /**
     * **بلا موقعٍ حيّ — عنوانُه المحفوظ الافتراضيّ مرّةً واحدة** (قرارُ المالك ٢٠٢٦-١١-٢٨)، **وبعد «نعم»
     * وحدَها**: لا إذنَ أو لا موقعَ فتصل الكابتنَ نقطةٌ أقربُ من لا شيء. بلا دقّة، ولا تتكرّر.
     */
    fun sendSavedAddress(scope: CoroutineScope, orderId: String) {
        scope.launch {
            try {
                val core = AppCore.get()
                val home = core.account.addresses().firstOrNull {
                    it.isDefault && (it.lat != 0.0 || it.lng != 0.0)
                } ?: return@launch
                CustomerApi(core.api).shareParty(orderId, home.lat, home.lng, null)
            } catch (e: CancellationException) {
                throw e
            } catch (_: Exception) {
                // **ما وصلت** — لا شيءَ يُعاد: العنوانُ احتياطٌ لمرّةٍ واحدة.
            }
        }
    }
}

/** **نصُّ سؤال البطاقة** — حسب النوع. */
@Composable
private fun partyQuestion(v: PartyView): String = when {
    v.kind == "amanah" && v.senderPhone.isNotEmpty() -> stringResource(R.string.party_q_amanah, v.senderPhone)
    v.kind == "amanah" -> stringResource(R.string.party_q_amanah_anon)
    v.kind == "delivery" && v.storeName.isNotEmpty() -> stringResource(R.string.party_q_delivery, v.storeName)
    else -> stringResource(R.string.party_q_order)
}

/**
 * **بطاقةُ «هل تريد استقبال…؟»** — العنوانُ والحالُ و«كابتن رحّال غو» (بلا اسم) والكودُ إن وُجد. **لا مبالغ.**
 */
@Composable
fun PartyDialog(
    view: PartyView,
    busy: Boolean,
    onYes: () -> Unit,
    onNo: () -> Unit,
    onDismiss: () -> Unit,
) {
    com.rahalgo.ui.RahalDialog(
        onDismissRequest = onDismiss,
        title = {
            Text(partyQuestion(view), fontWeight = FontWeight.Bold)
        },
        text = {
            Column(verticalArrangement = Arrangement.spacedBy(Rahal.space.sm)) {
                Text(
                    text = stringResource(R.string.party_status, statusText(view.status)),
                    color = Rahal.colors.ink,
                    style = Rahal.type.body,
                )
                if (view.captainAssigned) {
                    Text(
                        text = stringResource(R.string.party_captain),
                        color = Rahal.colors.ink,
                        style = Rahal.type.body,
                    )
                }
                if (view.code.isNotEmpty()) {
                    Text(
                        text = stringResource(R.string.party_code, view.code),
                        color = Rahal.colors.ink,
                        style = Rahal.type.body,
                        fontWeight = FontWeight.Bold,
                    )
                }
            }
        },
        confirmButton = {
            Column(
                verticalArrangement = Arrangement.spacedBy(Rahal.space.sm),
                modifier = Modifier.fillMaxWidth(),
            ) {
                RahalButton(onClick = onYes, enabled = !busy, tone = Tone.Success, modifier = Modifier.fillMaxWidth()) {
                    Text(stringResource(R.string.party_yes), fontWeight = FontWeight.Bold)
                }
                RahalButton(onClick = onNo, enabled = !busy, tone = Tone.Danger, modifier = Modifier.fillMaxWidth()) {
                    Text(stringResource(R.string.party_no))
                }
            }
        },
    )
}

/** **لافتةُ البيت** — «لديك أمانة بانتظار ردّك»؛ والضغطُ يفتح البطاقة. */
@Composable
fun PartyBanner(view: PartyView, modifier: Modifier = Modifier, onClick: () -> Unit) {
    RahalCard(
        modifier = modifier.fillMaxWidth().padding(horizontal = Rahal.space.md, vertical = Rahal.space.sm),
        tone = StatusTone.Warning,
        onClick = onClick,
    ) {
        Text(
            text = stringResource(
                if (PartyPolicy.isAmanah(view.kind)) R.string.party_banner_amanah else R.string.party_banner_order,
            ),
            color = Rahal.colors.ink,
            style = Rahal.type.body,
            fontWeight = FontWeight.Bold,
        )
    }
}
