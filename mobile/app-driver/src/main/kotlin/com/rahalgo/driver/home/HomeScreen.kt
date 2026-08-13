package com.rahalgo.driver.home

import androidx.compose.foundation.background
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
import com.rahalgo.design.BrandOrange
import com.rahalgo.design.BrandTeal
import com.rahalgo.design.InkMuted
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
                    Text(state.error, color = BrandOrange, textAlign = TextAlign.Center)
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
            Text(state.error, color = BrandOrange, modifier = Modifier.fillMaxWidth())
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

        Spacer(Modifier.height(20.dp))
        Text(
            text = stringResource(R.string.home_money),
            style = MaterialTheme.typography.titleMedium,
        )
        Spacer(Modifier.height(10.dp))
        MoneyRow(stringResource(R.string.home_balance), money(me.balance))
        // ══════════════════════════════════════════════════════════════
        // **النقد وسقفه في سطر واحد**
        // ══════════════════════════════════════════════════════════════
        //
        // **ومن بلغ سقفه لا يصله طلب نقدي** حتّى يسلّم للمكتب. **ورقم
        // وحده لا يقول أين هو من الحدّ** — فيُعرض الاثنان معا، ويحمرّ
        // حين يقترب.
        MoneyRow(
            label = stringResource(R.string.home_cash),
            // **و«من» لا شرطة مائلة**: السطر عربيّ يُقرأ من اليمين
            // وأرقامه لاتينيّة تُقرأ من اليسار، **والشرطة بينهما تنقلب
            // في العين** فلا يُعرف أيّ الرقمين الحدّ.
            value = if (me.cashLimit > 0) {
                stringResource(R.string.cash_of, grouped(me.cashHeld), money(me.cashLimit))
            } else {
                money(me.cashHeld)
            },
            warn = me.cashLimit > 0 && me.cashHeld >= me.cashLimit,
        )
        if (me.todayCompensated != 0L) {
            MoneyRow(stringResource(R.string.home_compensated), money(me.todayCompensated))
        }

        Spacer(Modifier.height(24.dp))
        TextButton(onClick = actions.logout, modifier = Modifier.fillMaxWidth()) {
            Icon(
                painter = painterResource(R.drawable.ic_logout),
                contentDescription = null,
                modifier = Modifier.size(18.dp),
            )
            Spacer(Modifier.size(8.dp))
            Text(stringResource(R.string.login_logout), color = InkMuted)
        }
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
            .background(Color(0xFFEFF4F6))
            .padding(16.dp),
    ) {
        Text(
            text = stringResource(R.string.offline_map_title),
            fontWeight = FontWeight.Bold,
            color = BrandTeal,
        )
        Spacer(Modifier.height(4.dp))
        Text(stringResource(R.string.offline_map_text), color = InkMuted)
        Spacer(Modifier.height(10.dp))
        if (downloading) {
            LinearProgressIndicator(
                progress = { progress / 100f },
                modifier = Modifier.fillMaxWidth(),
            )
            Spacer(Modifier.height(6.dp))
            Text(stringResource(R.string.offline_map_progress, progress), color = InkMuted)
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
            .background(Color(0xFFFFF4E5))
            .padding(16.dp),
    ) {
        Text(
            text = stringResource(R.string.loc_permission_title),
            fontWeight = FontWeight.Bold,
            color = BrandOrange,
        )
        Spacer(Modifier.height(4.dp))
        Text(stringResource(R.string.loc_permission_text), color = InkMuted)
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
            .background(if (on) BrandTeal else Color(0xFFEEF1F3))
            .padding(20.dp),
        horizontalAlignment = Alignment.CenterHorizontally,
    ) {
        Text(
            text = stringResource(if (on) R.string.shift_on else R.string.shift_off),
            style = MaterialTheme.typography.titleLarge,
            fontWeight = FontWeight.Bold,
            color = if (on) Color.White else InkMuted,
        )
        Spacer(Modifier.height(4.dp))
        Text(
            text = stringResource(
                if (on) R.string.shift_on_hint else R.string.shift_off_hint,
            ),
            color = if (on) Color.White.copy(alpha = 0.85f) else InkMuted,
            textAlign = TextAlign.Center,
        )
        Spacer(Modifier.height(16.dp))
        Button(
            onClick = { onToggle(!on) },
            enabled = !busy,
            colors = ButtonDefaults.buttonColors(
                containerColor = if (on) Color.White else BrandTeal,
                contentColor = if (on) BrandTeal else Color.White,
            ),
            modifier = Modifier.fillMaxWidth(),
        ) {
            if (busy) {
                CircularProgressIndicator(Modifier.size(18.dp), strokeWidth = 2.dp)
            } else {
                Text(stringResource(if (on) R.string.shift_end else R.string.shift_start))
            }
        }
        if (me.activeOrders > 0) {
            Spacer(Modifier.height(10.dp))
            Text(
                text = stringResource(
                    R.string.shift_active_orders,
                    grouped(me.activeOrders.toLong()),
                    grouped(me.maxActiveOrders),
                ),
                color = if (on) Color.White else InkMuted,
            )
        }
    }
}

@Composable
private fun Stat(label: String, value: String, modifier: Modifier = Modifier) {
    Column(
        modifier
            .clip(RoundedCornerShape(14.dp))
            .background(Color(0xFFF5F7F8))
            .padding(vertical = 14.dp, horizontal = 8.dp),
        horizontalAlignment = Alignment.CenterHorizontally,
    ) {
        Text(value, style = MaterialTheme.typography.titleLarge, fontWeight = FontWeight.Bold)
        Spacer(Modifier.height(2.dp))
        Text(label, color = InkMuted, style = MaterialTheme.typography.bodySmall)
    }
}

@Composable
private fun MoneyRow(label: String, value: String, warn: Boolean = false) {
    Row(
        Modifier.fillMaxWidth().padding(vertical = 8.dp),
        horizontalArrangement = Arrangement.SpaceBetween,
    ) {
        Text(label, color = InkMuted)
        Text(
            text = value,
            fontWeight = FontWeight.Bold,
            color = if (warn) BrandOrange else MaterialTheme.colorScheme.onSurface,
        )
    }
}

/** ما تعرضه اللوحة — **ولا تملكه هي.** */
data class HomeState(
    val me: DriverMe? = null,
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
