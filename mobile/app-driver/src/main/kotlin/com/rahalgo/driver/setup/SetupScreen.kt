package com.rahalgo.driver.setup

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.statusBarsPadding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.size
import androidx.compose.material3.Text
import com.rahalgo.driver.R
import com.rahalgo.driver.location.Readiness
import com.rahalgo.design.Rahal
import com.rahalgo.ui.RahalButton

/**
 * ══════════════════════════════════════════════════════════════════════
 * **إعدادُ أوّلِ دخول — طبقةُ عرضٍ رقيقةٌ فوق محرّك الجاهزيّة** (C، ٢٠٢٦-٠٩-٢٧)
 * ══════════════════════════════════════════════════════════════════════
 *
 * (قرارُ المالك: «طبقةٌ رقيقةٌ لا إعادةُ تصميم — أعِد استعمالَ آلة الحال
 *  المُثبتة، ولا تكرّر منطقَ الأذون».)
 *
 * # لا منطقَ أذونٍ هنا
 *
 * **القراراتُ كلُّها من `Readiness`** (مصدرُ الحقيقة الواحد)، **والطلباتُ
 * من مُطلِقات `SignedIn` نفسِها** (`ask`/`disclose`/`askNotify`) — تُمرَّر
 * ندءاً لا تُكتب هنا. **فلا إذنَ يُطلب مرّتين ولا آلةُ حالٍ ثانية.**
 *
 * # يُقاد بأوّل مانع
 *
 * **يعرض السببَ، ثمّ يطلب ما ينقص خطوةً خطوة** بترتيب `Readiness` نفسِه
 * (إذنٌ ← خدمةٌ ← خلفيّةٌ ← إشعار)، **ويُكمَل حين يصير الموقعُ منتِجاً**
 * (`canWork`) فيبدأ تحضيرُ الخريطة تلقائيّاً — **بلا إدارةِ خرائطَ يدويّة.**
 *
 * # ولا يُعرض ثانيةً
 *
 * **يُستدعى ما دام العلمُ المخزَّنُ غيرَ مضبوط** (`SignedIn`)، **فإذا اكتمل
 * ضُبط ولم يعُد يظهر** — وإن فسد الإذنُ لاحقاً تولّته الواجهةُ التفاعليّة
 * القائمةُ (`LocationCard`)، لا هذا المعالج.
 */
@Composable
fun SetupWizard(
    readiness: Readiness.State,
    batteryExempt: Boolean,
    onGrantLocation: () -> Unit,
    onEnableGps: () -> Unit,
    onGrantBackground: () -> Unit,
    onGrantNotifications: () -> Unit,
    onGrantBattery: () -> Unit,
    onOpenAppSettings: () -> Unit,
    onStartMapPrep: () -> Unit,
    onDone: () -> Unit,
) {
    val canWork = Readiness.canWork(readiness)
    // **وتحضيرُ الخريطة يبدأ تلقائيّاً حالَ صار الموقعُ منتِجاً** — بلا زرّ.
    LaunchedEffect(canWork) { if (canWork) onStartMapPrep() }

    Column(
        Modifier
            .fillMaxSize()
            .background(Rahal.colors.surface)
            .statusBarsPadding()
            .verticalScroll(rememberScrollState())
            .padding(24.dp),
    ) {
        Spacer(Modifier.height(24.dp))
        Text(
            text = stringResource(R.string.setup_title),
            style = androidx.compose.material3.MaterialTheme.typography.headlineSmall,
            fontWeight = FontWeight.Bold,
            color = Rahal.colors.brand,
        )
        Spacer(Modifier.height(8.dp))
        Text(stringResource(R.string.setup_why), color = Rahal.colors.inkMuted)
        Spacer(Modifier.height(20.dp))

        // ── خطواتٌ بترتيب `Readiness` — كلٌّ بحالتها من القائمة نفسِها ──
        // **ورقمٌ أو علامةُ صحّ لكلّ خطوة، والحاليّةُ مظلّلة، وفوقها «خلصت X من Y»**
        // (تدقيق ٢٠٢٦-١٠-٢٧) — كانت نقاطاً تفترق باللون وحدَه.
        // **وإعفاءُ البطّاريّة — «اعمل في الخلفية»** — خطوةُ تهيئةٍ لا بطاقةَ
        // رئيسيّةٍ دائمة (قرارُ المالك ٢٠٢٦-٠٩-٢٨). **استرشاديٌّ لا يحجب المتابعة.**
        val steps = listOf(
            stringResource(R.string.setup_step_location) to
                !readiness.blockers.contains(Readiness.Blocker.LOCATION_PERMISSION_REQUIRED),
            stringResource(R.string.setup_step_gps) to
                (!readiness.blockers.contains(Readiness.Blocker.LOCATION_SERVICE_OFF) &&
                    !readiness.blockers.contains(Readiness.Blocker.LOCATION_PERMISSION_REQUIRED)),
            stringResource(R.string.setup_step_background) to
                !readiness.blockers.contains(Readiness.Blocker.BACKGROUND_LOCATION_REQUIRED),
            stringResource(R.string.setup_step_notifications) to
                !readiness.blockers.contains(Readiness.Blocker.NOTIFICATION_PERMISSION_REQUIRED),
            stringResource(R.string.setup_step_battery) to batteryExempt,
        )
        val doneCount = steps.count { it.second }
        val current = steps.indexOfFirst { !it.second }
        Text(
            text = stringResource(R.string.setup_progress, doneCount, steps.size),
            color = Rahal.colors.ink,
            fontWeight = FontWeight.Bold,
        )
        Spacer(Modifier.height(6.dp))
        com.rahalgo.ui.Bar(ratio = doneCount.toFloat() / steps.size, color = Rahal.colors.success)
        Spacer(Modifier.height(14.dp))
        steps.forEachIndexed { i, (label, done) ->
            StepRow(number = i + 1, done = done, current = i == current, label = label)
        }

        Spacer(Modifier.height(20.dp))

        // ── الفعلُ التالي: أوّلُ مانعٍ في الحال يقرّر الزرَّ ونصَّه ──
        val next = readiness.first
        if (next != null) {
            val (action, labelRes) = when (next) {
                Readiness.Blocker.LOCATION_PERMISSION_REQUIRED ->
                    onGrantLocation to R.string.setup_grant_location
                Readiness.Blocker.LOCATION_SERVICE_OFF ->
                    onEnableGps to R.string.setup_enable_gps
                Readiness.Blocker.BACKGROUND_LOCATION_REQUIRED ->
                    onGrantBackground to R.string.setup_grant_background
                Readiness.Blocker.NOTIFICATION_PERMISSION_REQUIRED ->
                    onGrantNotifications to R.string.setup_grant_notifications
            }
            RahalButton(onClick = action, modifier = Modifier.fillMaxWidth()) {
                Text(stringResource(labelRes))
            }
            Spacer(Modifier.height(10.dp))
            // **ورفضٌ نهائيٌّ لا يُصلحه طلبٌ ثانٍ** — فبابُ الإعدادات دائماً.
            Text(
                text = stringResource(R.string.setup_recovery_hint),
                color = Rahal.colors.inkMuted,
                style = androidx.compose.material3.MaterialTheme.typography.bodySmall,
            )
            // **وبابُ الإعدادات رابطٌ صغير** — لا زرٌّ برتقاليٌّ ممتلئٌ ينازع الأساسيّ.
            com.rahalgo.ui.RahalTextButton(onClick = onOpenAppSettings, modifier = Modifier.fillMaxWidth()) {
                Text(stringResource(R.string.setup_open_settings))
            }
        }

        // ── إعفاءُ البطّاريّة يُطلَب حين يصير الموقعُ منتِجاً، قبل المتابعة ──
        // **ولا يحجبها**: استرشاديٌّ (لا يدخل `canWork`)، فيُعرض زرُّه ما لم
        // يُمنَح، ويختفي حين يُمنَح — **بلا «لاحقاً» دائمةٍ على الرئيسيّة.**
        if (canWork && !batteryExempt) {
            Spacer(Modifier.height(16.dp))
            RahalButton(onClick = onGrantBattery, modifier = Modifier.fillMaxWidth()) {
                Text(stringResource(R.string.setup_grant_battery))
            }
        }

        // ── المتابعةُ حين يصير الموقعُ منتِجاً — والباقي تحرسه الجاهزيّة ──
        if (canWork) {
            Spacer(Modifier.height(16.dp))
            RahalButton(onClick = onDone, modifier = Modifier.fillMaxWidth()) {
                Text(stringResource(R.string.setup_continue))
            }
        }
    }
}

@Composable
private fun StepRow(number: Int, done: Boolean, current: Boolean, label: String) {
    Row(
        Modifier
            .fillMaxWidth()
            .clip(Rahal.shape.sm)
            .background(if (current) Rahal.colors.brand.copy(alpha = 0.08f) else androidx.compose.ui.graphics.Color.Transparent)
            .padding(horizontal = 8.dp, vertical = 8.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Box(
            Modifier
                .size(26.dp)
                .background(
                    when {
                        done -> Rahal.colors.success
                        current -> Rahal.colors.brand
                        else -> Rahal.colors.field
                    },
                    CircleShape,
                ),
            contentAlignment = Alignment.Center,
        ) {
            if (done) {
                androidx.compose.material3.Icon(
                    painter = androidx.compose.ui.res.painterResource(R.drawable.ic_check_circle),
                    contentDescription = null,
                    tint = androidx.compose.ui.graphics.Color.White,
                    modifier = Modifier.size(18.dp),
                )
            } else {
                Text(
                    text = com.rahalgo.ui.grouped(number.toLong()),
                    color = if (current) Rahal.colors.onBrand else Rahal.colors.inkMuted,
                    fontWeight = FontWeight.Bold,
                    style = androidx.compose.material3.MaterialTheme.typography.labelMedium,
                )
            }
        }
        Spacer(Modifier.width(10.dp))
        Text(
            label,
            color = when {
                done -> Rahal.colors.success
                current -> Rahal.colors.ink
                else -> Rahal.colors.inkMuted
            },
            fontWeight = if (current) FontWeight.Bold else FontWeight.Normal,
        )
    }
    Spacer(Modifier.height(6.dp))
}
