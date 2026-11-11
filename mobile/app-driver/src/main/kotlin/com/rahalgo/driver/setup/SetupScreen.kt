package com.rahalgo.driver.setup

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.heightIn
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
            .background(Rahal.colors.canvas)
            .statusBarsPadding()
            .verticalScroll(rememberScrollState())
            .padding(Rahal.space.gutter),
    ) {
        Spacer(Modifier.height(Rahal.space.xl))
        Text(
            text = stringResource(R.string.setup_title),
            style = Rahal.type.headline,
            color = Rahal.colors.ink,
        )
        Spacer(Modifier.height(Rahal.space.sm))
        Text(stringResource(R.string.setup_why), color = Rahal.colors.inkMuted, style = Rahal.type.body)
        Spacer(Modifier.height(Rahal.space.lg))

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
        // **الخطواتُ في بطاقةٍ واحدةٍ بحدود** (إعادةُ التصميم ٢٠٢٦-١١-١١) — العدّادُ وشريطُه رأسُها.
        com.rahalgo.ui.RahalCard(variant = com.rahalgo.ui.CardVariant.Outlined, padding = Rahal.space.lg) {
            Text(
                text = stringResource(R.string.setup_progress, doneCount, steps.size),
                color = Rahal.colors.ink,
                style = Rahal.type.cardTitle,
            )
            Spacer(Modifier.height(Rahal.space.sm))
            com.rahalgo.ui.Bar(ratio = doneCount.toFloat() / steps.size, color = Rahal.colors.success)
            Spacer(Modifier.height(Rahal.space.md))
            steps.forEachIndexed { i, (label, done) ->
                StepRow(number = i + 1, done = done, current = i == current, label = label)
            }
        }

        Spacer(Modifier.height(Rahal.space.lg))

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
                Text(stringResource(labelRes), style = Rahal.type.button)
            }
            Spacer(Modifier.height(Rahal.space.md))
            // **ورفضٌ نهائيٌّ لا يُصلحه طلبٌ ثانٍ** — فبابُ الإعدادات دائماً.
            Text(
                text = stringResource(R.string.setup_recovery_hint),
                color = Rahal.colors.inkMuted,
                style = Rahal.type.caption,
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
            Spacer(Modifier.height(Rahal.space.lg))
            // **ثانويٌّ بحدود** — استرشاديٌّ لا يحجب، والأساسيُّ «متابعة» تحته.
            com.rahalgo.ui.RahalOutlineButton(onClick = onGrantBattery, modifier = Modifier.fillMaxWidth()) {
                Text(stringResource(R.string.setup_grant_battery), style = Rahal.type.button)
            }
        }

        // ── المتابعةُ حين يصير الموقعُ منتِجاً — والباقي تحرسه الجاهزيّة ──
        if (canWork) {
            Spacer(Modifier.height(Rahal.space.lg))
            RahalButton(onClick = onDone, modifier = Modifier.fillMaxWidth()) {
                Text(stringResource(R.string.setup_continue), style = Rahal.type.button)
            }
        }
    }
}

@Composable
private fun StepRow(number: Int, done: Boolean, current: Boolean, label: String) {
    Row(
        Modifier
            .fillMaxWidth()
            .clip(Rahal.shape.md)
            .background(if (current) Rahal.colors.brand.copy(alpha = 0.08f) else androidx.compose.ui.graphics.Color.Transparent)
            .heightIn(min = Rahal.size.touch)
            .padding(horizontal = Rahal.space.sm, vertical = Rahal.space.xs),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Box(
            Modifier
                .size(Rahal.icon.lg)
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
                    tint = Rahal.colors.onBrand,
                    modifier = Modifier.size(Rahal.icon.sm),
                )
            } else {
                Text(
                    text = com.rahalgo.ui.grouped(number.toLong()),
                    color = if (current) Rahal.colors.onBrand else Rahal.colors.inkMuted,
                    fontWeight = FontWeight.Bold,
                    style = Rahal.type.label,
                )
            }
        }
        Spacer(Modifier.width(Rahal.space.md))
        Text(
            label,
            color = when {
                done -> Rahal.colors.ink
                current -> Rahal.colors.ink
                else -> Rahal.colors.inkMuted
            },
            style = if (current) Rahal.type.bodyStrong else Rahal.type.body,
        )
    }
    Spacer(Modifier.height(Rahal.space.xs))
}
