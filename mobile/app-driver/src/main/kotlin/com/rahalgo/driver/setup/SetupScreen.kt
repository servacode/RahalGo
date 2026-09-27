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
    onGrantLocation: () -> Unit,
    onEnableGps: () -> Unit,
    onGrantBackground: () -> Unit,
    onGrantNotifications: () -> Unit,
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
        StepRow(
            done = !readiness.blockers.contains(Readiness.Blocker.LOCATION_PERMISSION_REQUIRED),
            label = stringResource(R.string.setup_step_location),
        )
        StepRow(
            done = !readiness.blockers.contains(Readiness.Blocker.LOCATION_SERVICE_OFF) &&
                !readiness.blockers.contains(Readiness.Blocker.LOCATION_PERMISSION_REQUIRED),
            label = stringResource(R.string.setup_step_gps),
        )
        StepRow(
            done = !readiness.blockers.contains(Readiness.Blocker.BACKGROUND_LOCATION_REQUIRED),
            label = stringResource(R.string.setup_step_background),
        )
        StepRow(
            done = !readiness.blockers.contains(Readiness.Blocker.NOTIFICATION_PERMISSION_REQUIRED),
            label = stringResource(R.string.setup_step_notifications),
        )

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
            Spacer(Modifier.height(6.dp))
            RahalButton(
                onClick = onOpenAppSettings,
                modifier = Modifier.fillMaxWidth(),
                tone = com.rahalgo.ui.Tone.Accent,
            ) {
                Text(stringResource(R.string.setup_open_settings))
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
private fun StepRow(done: Boolean, label: String) {
    Row(verticalAlignment = Alignment.CenterVertically) {
        Box(
            Modifier
                .size(18.dp)
                .background(
                    if (done) Rahal.colors.brand else Rahal.colors.field,
                    CircleShape,
                ),
        )
        Spacer(Modifier.width(10.dp))
        Text(label, color = if (done) Rahal.colors.brand else Rahal.colors.inkMuted)
    }
    Spacer(Modifier.height(10.dp))
}
