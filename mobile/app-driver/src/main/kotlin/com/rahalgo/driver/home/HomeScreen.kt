package com.rahalgo.driver.home

import androidx.compose.foundation.background
import com.rahalgo.driver.ui.Bar
import androidx.compose.material3.SwitchDefaults
import androidx.compose.material3.Switch
import com.rahalgo.shared.model.IncentivesPayload
import com.rahalgo.design.Rahal
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.statusBarsPadding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import com.rahalgo.driver.trip.OfflineMap
import androidx.compose.ui.platform.LocalContext
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.material3.LinearProgressIndicator
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import com.rahalgo.driver.R
import com.rahalgo.driver.ui.grouped
import com.rahalgo.driver.ui.money
import com.rahalgo.shared.model.DriverMe

/**
 * ══════════════════════════════════════════════════════════════════════
 * **لوحة السائق — الوردية أوّلا**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (خطوة البناء الرابعة، قرار المالك ٢٠٢٦-٠٨-١١.)
 *
 * # لماذا الوردية في أعلى الشاشة وأكبر شيء فيها
 *
 * **من ليس على وردية لا يصله طلب** — وهذا كلّ شيء بالنسبة له. **وسائق
 * ينتظر طلبا لا يأتي لأنّ ورديته مغلقة** هو أسوأ ما يقع في يومه، ولا شيء
 * في الشاشة يقول له لماذا.
 *
 * **فتقول الشاشة حاله بلونها كلّها** لا بسطر صغير: أخضر يعمل، رمادي
 * واقف.
 *
 * # وأرقام اليوم لا أرقام الشهر
 *
 * **السائق يسأل: كم سلّمت اليوم وكم كسبت؟** — والمجاميع الكبيرة تخصّ
 * المكتب. **ومن عرض الشهر أضاع الرقم الذي يُسأل عنه في الشارع.**
 *
 * # ولا منطق هنا
 *
 * (`GROUND-RULES.md` §7.2 البند ٥.) **الشاشة تعرض وترسل** —
 * والنداءات في `HomeViewModel`.
 */
@Composable
fun HomeScreen(state: HomeState, actions: HomeActions) {
    if (state.me == null) {
        Box(Modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
            if (state.error.isEmpty()) {
                CircularProgressIndicator()
            } else {
                // **وأوّل نداء قد يسقط والشاشة فارغة** — فلا يبقى صاحبها
                // أمام بياض بلا سبب ولا زرّ.
                Column(horizontalAlignment = Alignment.CenterHorizontally) {
                    Text(state.error, color = Rahal.colors.accent, textAlign = TextAlign.Center)
                    TextButton(onClick = actions.refresh) {
                        Text(stringResource(R.string.home_retry))
                    }
                }
            }
        }
        return
    }

    val me = state.me
    val context = LocalContext.current
    // **ويُفحص عند كلّ فتح** — قد يكون نزّلها ثمّ مسح بيانات التطبيق.
    LaunchedEffect(Unit) { OfflineMap.check(context) }
    Column(
        Modifier
            .fillMaxSize()
            // **وتحت شريط الحالة لا خلفه** — الشاشة تُرسم من حافة إلى
            // حافة. **وشريط النظام السفليّ يحسبه `Scaffold` مرّة**،
            // فلو أُضيف هنا `safeDrawing` لحُسب مرّتين وارتفع المحتوى.
            .statusBarsPadding()
            .verticalScroll(rememberScrollState())
            .padding(horizontal = 20.dp),
    ) {
        Spacer(Modifier.height(28.dp))
        Text(
            text = stringResource(R.string.login_welcome, me.fullName),
            style = MaterialTheme.typography.headlineSmall,
        )
        Spacer(Modifier.height(20.dp))

        // ══════════════════════════════════════════════════════════════
        // **الموقع مطفأ يُقال فوق كلّ شيء**
        // ══════════════════════════════════════════════════════════════
        //
        // **وهو أوّل ما يُسأل عنه المكتب**: «لماذا لا تصلني طلبات؟»
        // — لأنّ المحرّك لا يعرف أين هو، **فلا يحسب مسافة ولا يعرض
        // عليه أقرب طلب.** والسائق لا يرى من ذلك شيئا.
        if (!state.locationOn) {
            LocationCard(onEnable = actions.enableLocation)
            Spacer(Modifier.height(14.dp))
        }

        // ══════════════════════════════════════════════════════════════
        // **خريطة المدينة في الجهاز**
        // ══════════════════════════════════════════════════════════════
        //
        // **وتُعرض ما لم تُنزَّل** — ولا تُنزَّل بنفسها: عشرات الميغابايت
        // من حزمة السائق **قرارُه هو.**
        if (!OfflineMap.ready) {
            OfflineCard(
                progress = OfflineMap.progress,
                downloading = OfflineMap.downloading,
                onDownload = { OfflineMap.download(context) },
            )
            Spacer(Modifier.height(14.dp))
        }

        ShiftCard(me = me, busy = state.busy, onToggle = actions.toggleShift)

        if (state.error.isNotEmpty()) {
            Spacer(Modifier.height(12.dp))
            Text(state.error, color = Rahal.colors.accent, modifier = Modifier.fillMaxWidth())
        }

        Spacer(Modifier.height(20.dp))
        Text(
            text = stringResource(R.string.home_today),
            style = MaterialTheme.typography.titleMedium,
        )
        Spacer(Modifier.height(10.dp))
        Row(horizontalArrangement = Arrangement.spacedBy(10.dp)) {
            Stat(
                label = stringResource(R.string.home_delivered),
                value = grouped(me.todayDelivered.toLong()),
                modifier = Modifier.weight(1f),
            )
            Stat(
                label = stringResource(R.string.home_failed),
                value = grouped(me.todayFailed.toLong()),
                modifier = Modifier.weight(1f),
            )
            Stat(
                label = stringResource(R.string.home_earned),
                value = grouped(me.todayEarned),
                modifier = Modifier.weight(1f),
            )
        }

        // ══════════════════════════════════════════════════════════════
        // **ولا سطرَ مالٍ هنا — رصيدُه في الشريط فوقه**
        // ══════════════════════════════════════════════════════════════
        //
        // (قرارُ المالك ٢٠٢٦-٠٨-١٣: «المالُ نحذفه، ما يلزم — موجودٌ فوق
        //  أساساً».)
        //
        // **ورقمٌ يُعاد في شاشةٍ واحدةٍ مرّتين يُقرأ رقمين** — فيسأل
        // صاحبُه أيُّهما الصحيح.

        // ══════════════════════════════════════════════════════════════
        // **وصندوقُه شريطٌ لا سطر**
        // ══════════════════════════════════════════════════════════════
        //
        // (قرارُ المالك ٢٠٢٦-٠٨-١٣: «نحطّ شريطَ صندوقي، شقد في ذمّة
        //  السائق».)
        //
        // **والسقفُ ليس زينة**: من بلغه لا يُعرض عليه طلبٌ نقديّ، **فيقف
        // عملُه ولا يعرف لماذا.** **وشريطٌ يُرى قبل أن يُقرأ رقم.**
        if (me.cashLimit > 0) {
            Spacer(Modifier.height(22.dp))
            CashBar(held = me.cashHeld, limit = me.cashLimit)
        }

        // ══════════════════════════════════════════════════════════════
        // **وهدفُه بعده — وجائزتُه معه**
        // ══════════════════════════════════════════════════════════════
        //
        // (قرارُ المالك ٢٠٢٦-٠٨-١٣: «بعدها نحطّ شريطَ الهدف وقيمةَ
        //  المكافأة عند تحقيق الهدف».)
        //
        // **وحافزٌ لا يُرى لا يحفّز** — ومن لا يعرف أنّه على بُعد ثلاثةِ
        // طلباتٍ من مكافأةٍ لا يسعى إليها.
        val st = state.goal?.standing
        if (st != null && st.target > 0) {
            Spacer(Modifier.height(18.dp))
            GoalBar(
                done = st.done,
                target = st.target,
                reached = st.reached,
                reward = state.goal?.targetReward ?: 0,
            )
        }

        // **والخروجُ انتقل إلى أسفل القائمة الجانبيّة** — (قرارُ المالك
        // ٢٠٢٦-٠٨-١٣). **وموضعان لفعلٍ واحدٍ يجعلان أحدَهما يُنسى**،
        // ولوحةُ العمل ليست موضعَ فعلٍ يُنهي الجلسة.
        Spacer(Modifier.height(28.dp))
    }
}

/**
 * **بطاقة تنزيل خريطة المدينة.**
 *
 * **وتقول ماذا يكسب لا ما نطلبه**: «تشتغل بلا إنترنت» أوقع من «نزّل
 * البيانات».
 */
@Composable
private fun OfflineCard(progress: Int, downloading: Boolean, onDownload: () -> Unit) {
    Column(
        Modifier
            .fillMaxWidth()
            .clip(RoundedCornerShape(16.dp))
            .background(Rahal.colors.surface)
            .padding(16.dp),
    ) {
        Text(
            text = stringResource(R.string.offline_map_title),
            fontWeight = FontWeight.Bold,
            color = Rahal.colors.brand,
        )
        Spacer(Modifier.height(4.dp))
        Text(stringResource(R.string.offline_map_text), color = Rahal.colors.inkMuted)
        Spacer(Modifier.height(10.dp))
        if (downloading) {
            LinearProgressIndicator(
                progress = { progress / 100f },
                modifier = Modifier.fillMaxWidth(),
            )
            Spacer(Modifier.height(6.dp))
            Text(stringResource(R.string.offline_map_progress, progress), color = Rahal.colors.inkMuted)
        } else {
            Button(onClick = onDownload, modifier = Modifier.fillMaxWidth()) {
                // **ومن وقف في نصفه يُقال له «أكمل» لا «نزّل»** — الأوّل
                // يقول إنّ ما مضى محفوظ.
                Text(
                    stringResource(
                        if (progress in 1..99) R.string.offline_map_resume
                        else R.string.offline_map_button,
                    ),
                )
            }
        }
    }
}

/**
 * **بطاقة تفعيل الموقع.**
 *
 * **وتقول ماذا يخسر لا ماذا يريد النظام**: «لا تعرف كم يبعد المتجر»
 * أوقع من «التطبيق يحتاج إذن الموقع».
 */
@Composable
private fun LocationCard(onEnable: () -> Unit) {
    Column(
        Modifier
            .fillMaxWidth()
            .clip(RoundedCornerShape(16.dp))
            .background(Rahal.colors.warnTint)
            .padding(16.dp),
    ) {
        Text(
            text = stringResource(R.string.loc_permission_title),
            fontWeight = FontWeight.Bold,
            color = Rahal.colors.accent,
        )
        Spacer(Modifier.height(4.dp))
        Text(stringResource(R.string.loc_permission_text), color = Rahal.colors.inkMuted)
        Spacer(Modifier.height(10.dp))
        Button(onClick = onEnable, modifier = Modifier.fillMaxWidth()) {
            Text(stringResource(R.string.loc_permission_button))
        }
    }
}

/**
 * **بطاقة الوردية.**
 *
 * **والزرّ يقول ما سيحدث لا ما هو حادث**: «ابدأ الوردية» حين تكون مغلقة.
 * **ومن كتب حاله على الزرّ** جعل نصفهم يضغط ليصل إلى ما هو فيه أصلا.
 */
@Composable
private fun ShiftCard(me: DriverMe, busy: Boolean, onToggle: (Boolean) -> Unit) {
    val on = me.onShift
    Column(
        Modifier
            .fillMaxWidth()
            .clip(RoundedCornerShape(18.dp))
            .background(if (on) Rahal.colors.brand else Rahal.colors.field)
            .padding(20.dp),
        horizontalAlignment = Alignment.CenterHorizontally,
    ) {
        // ══════════════════════════════════════════════════════════════
        // **مفتاحٌ واحدٌ — حالٌ وفعلٌ معا**
        // ══════════════════════════════════════════════════════════════
        //
        // (قرارُ المالك ٢٠٢٦-٠٨-١٣: «نخلّيه زرّاً ذكيّاً بالعمل وخارج
        //  العمل، وتحته يُكتب: الآن تستقبل الطلبات · لا يمكنك استقبال
        //  الطلبات أنت خارج العمل».)
        //
        // **وكانت البطاقةُ تقول شيئين متضادّين**: أعلاها الحالُ «وردية
        // مفتوحة» وأسفلها الفعلُ «أنهِ الوردية» — **ومن قرأ بسرعةٍ وهو
        // يقود لا يعرف أيُّهما حالُه.**
        //
        // **والمفتاحُ يقول الحالَ بموضعه ويقبل الفعلَ بلمسته** — ولا
        // كلمتين متضادّتين في بطاقةٍ واحدة.
        Row(
            Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.SpaceBetween,
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Text(
                text = stringResource(if (on) R.string.shift_on else R.string.shift_off),
                style = MaterialTheme.typography.titleLarge,
                fontWeight = FontWeight.Bold,
                color = if (on) Rahal.colors.onBrand else Rahal.colors.inkMuted,
            )
            if (busy) {
                CircularProgressIndicator(Modifier.size(22.dp), strokeWidth = 2.dp)
            } else {
                Switch(
                    checked = on,
                    onCheckedChange = onToggle,
                    colors = SwitchDefaults.colors(
                        checkedThumbColor = Rahal.colors.onBrand,
                        checkedTrackColor = Rahal.colors.success,
                        uncheckedThumbColor = Rahal.colors.canvas,
                        uncheckedTrackColor = Rahal.colors.inkMuted,
                    ),
                )
            }
        }
        Spacer(Modifier.height(6.dp))
        // **وما يترتّب عليه تحته** — لا يُترك ليُستنتج من كلمتين.
        Text(
            text = stringResource(
                if (on) R.string.shift_on_hint else R.string.shift_off_hint,
            ),
            color = if (on) Rahal.colors.onBrand.copy(alpha = 0.85f) else Rahal.colors.inkMuted,
        )
        if (me.activeOrders > 0) {
            Spacer(Modifier.height(10.dp))
            Text(
                text = stringResource(
                    R.string.shift_active_orders,
                    grouped(me.activeOrders.toLong()),
                    grouped(me.maxActiveOrders),
                ),
                color = if (on) Rahal.colors.onBrand else Rahal.colors.inkMuted,
            )
        }
    }
}

@Composable
private fun Stat(label: String, value: String, modifier: Modifier = Modifier) {
    Column(
        modifier
            .clip(RoundedCornerShape(14.dp))
            .background(Rahal.colors.surface)
            .padding(vertical = 14.dp, horizontal = 8.dp),
        horizontalAlignment = Alignment.CenterHorizontally,
    ) {
        Text(value, style = MaterialTheme.typography.titleLarge, fontWeight = FontWeight.Bold)
        Spacer(Modifier.height(2.dp))
        Text(label, color = Rahal.colors.inkMuted, style = MaterialTheme.typography.bodySmall)
    }
}

@Composable
private fun MoneyRow(label: String, value: String, warn: Boolean = false) {
    Row(
        Modifier.fillMaxWidth().padding(vertical = 8.dp),
        horizontalArrangement = Arrangement.SpaceBetween,
    ) {
        Text(label, color = Rahal.colors.inkMuted)
        Text(
            text = value,
            fontWeight = FontWeight.Bold,
            color = if (warn) Rahal.colors.accent else MaterialTheme.colorScheme.onSurface,
        )
    }
}

/** ما تعرضه اللوحة — **ولا تملكه هي.** */
data class HomeState(
    val me: DriverMe? = null,
    /**
     * **هدفُه ومكافأتُه** — (قرارُ المالك ٢٠٢٦-٠٨-١٣: «نحطّ شريطَ الهدف
     *  وقيمةَ المكافأة عند تحقيق الهدف»).
     *
     * **وفشلُ جلبه لا يُسقط اللوحة** — يُطوى الشريطُ ويبقى ما سواه:
     * **مفتاحُ العمل أهمُّ ما فيها، ولا يُحجب لأجل رقمٍ لم يصل.**
     */
    val goal: IncentivesPayload? = null,
    /** هل إذن الموقع ممنوح؟ — **وبدونه لا مسافة ولا أقرب طلب.** */
    val locationOn: Boolean = true,
    val busy: Boolean = false,
    val error: String = "",
)

data class HomeActions(
    val toggleShift: (Boolean) -> Unit,
    val enableLocation: () -> Unit,
    val refresh: () -> Unit,
    val logout: () -> Unit,
)

/**
 * **شريطُ الصندوق** — ما في ذمّته وكم بقي قبل سقفه.
 *
 * **ويُقال ما يبقى لا ما مضى**: «يبقى ٣٫١٠٠» جوابُ سؤاله، **و«١٬٩٠٠ من
 * ٥٬٠٠٠» كسرٌ يُحسب في الرأس.**
 */
@Composable
private fun CashBar(held: Long, limit: Long) {
    val left = limit - held
    val ratio = if (limit > 0) held.toFloat() / limit else 0f
    Row(
        Modifier.fillMaxWidth(),
        horizontalArrangement = Arrangement.SpaceBetween,
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Text(
            text = stringResource(R.string.home_cash_bar),
            style = MaterialTheme.typography.titleMedium,
        )
        Text(
            text = money(held),
            fontWeight = FontWeight.Bold,
            color = if (ratio >= 1f) Rahal.colors.danger else Rahal.colors.accent,
        )
    }
    Spacer(Modifier.height(8.dp))
    Bar(
        ratio = ratio,
        color = when {
            ratio >= 1f -> Rahal.colors.danger
            ratio > 0.8f -> Rahal.colors.accent
            else -> Rahal.colors.success
        },
    )
    Spacer(Modifier.height(6.dp))
    Text(
        text = if (left > 0) {
            stringResource(R.string.home_cash_left, money(left))
        } else {
            stringResource(R.string.home_cash_over)
        },
        color = if (left > 0) Rahal.colors.inkMuted else Rahal.colors.danger,
        style = MaterialTheme.typography.bodySmall,
    )
}

/**
 * **شريطُ الهدف** — ما أنجزه من شهره، وما ينتظره إن بلغه.
 *
 * **والجائزةُ تُقال قبل أن تُنال** — وشاشةٌ تقول «٨ من ٥٠» ولا تقول ماذا
 * بعدها **تطلب جهداً بلا وعد.** **وصفرٌ يُخفيها**: لا يُعرَض وعدٌ بلا
 * مبلغ.
 */
@Composable
private fun GoalBar(done: Int, target: Int, reached: Boolean, reward: Long) {
    Row(
        Modifier.fillMaxWidth(),
        horizontalArrangement = Arrangement.SpaceBetween,
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Text(
            text = stringResource(R.string.home_target),
            style = MaterialTheme.typography.titleMedium,
        )
        Text(
            text = grouped(done.toLong()) + " / " + grouped(target.toLong()),
            fontWeight = FontWeight.Bold,
            color = if (reached) Rahal.colors.success else Rahal.colors.brand,
        )
    }
    Spacer(Modifier.height(8.dp))
    Bar(
        ratio = done.toFloat() / target,
        color = if (reached) Rahal.colors.success else Rahal.colors.brand,
    )
    Spacer(Modifier.height(6.dp))
    Row(
        Modifier.fillMaxWidth(),
        horizontalArrangement = Arrangement.SpaceBetween,
    ) {
        if (!reached) {
            Text(
                text = stringResource(
                    R.string.home_target_left,
                    grouped((target - done).toLong()),
                ),
                color = Rahal.colors.inkMuted,
                style = MaterialTheme.typography.bodySmall,
            )
        }
        if (reward > 0) {
            Text(
                text = stringResource(
                    if (reached) R.string.home_target_got else R.string.home_target_reward,
                    money(reward),
                ),
                color = if (reached) Rahal.colors.success else Rahal.colors.accent,
                fontWeight = FontWeight.Medium,
                style = MaterialTheme.typography.bodySmall,
            )
        }
    }
}
