package com.rahalgo.driver.trip

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.defaultMinSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.semantics.clearAndSetSemantics
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.selected
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.rahalgo.design.Rahal
import com.rahalgo.design.RahalShapeTokens
import com.rahalgo.design.RahalSpaceTokens
import com.rahalgo.driver.R
import com.rahalgo.navigation.MapRoutePalette
import com.rahalgo.navigation.RouteChoiceUi
import com.rahalgo.navigation.RouteMetricText
import com.rahalgo.navigation.RouteOption

/**
 * ══════════════════════════════════════════════════════════════════
 * **لوحةُ اختيار المسار — صغيرةٌ فوق عناصر التحكّم**
 * ══════════════════════════════════════════════════════════════════
 *
 * (إغلاقُ واجهة ٧، قرارُ المالك ٢٠٢٦-٠٨-٢١، البنود ٥ إلى ١١ و٢٩.)
 *
 * **أمرُ المالك نصّاً**: «ليس BottomSheet كبيرة · ليس Modal · ليس صفحة
 * جديدة… عنصرًا صغيرًا فوق Controls الحالية».
 *
 * # **ولا تظهر بلا فائدة** (البند ٥)
 *
 * **لا بطاقةً فارغةً ولا «لا توجد طرق بديلة»** — **شاشةُ السائق ضيّقة
 * وهو يقود**، ومساحةٌ تُؤخذ بلا معلومةٍ خسارةٌ صافية.
 *
 * # **وضغطةٌ لا تعتمد** (البند ١١)
 *
 * **البديلُ يدخل معاينةً**، ثمّ يظهر زرُّ **«اعتماد المسار»**.
 * **فلمسةٌ خاطئةٌ على مِقودٍ لا تبدّل ملاحةً وصوتاً ومناورات.**
 *
 * # **وأرقامٌ لا صفات** (البند ٨)
 *
 * **لا «الأسرع» ولا «الأقصر»** — فرقان بالمتر والدقيقة: «أقصرُ بـ٤٫٥
 * كم · أطولُ زمناً بـ١٢ د». **ووزنُ محرّكنا `routability` لا
 * `duration`**، فالصفةُ ادّعاءٌ لا يُثبت.
 */
@Composable
fun RouteChoicePanel(
    ui: RouteChoiceUi,
    onSelect: (String) -> Unit,
    onConfirm: () -> Unit,
    onCancel: () -> Unit,
    modifier: Modifier = Modifier,
) {
    val choices = ui.choicesOrNull ?: return
    if (ui is RouteChoiceUi.Hidden || ui is RouteChoiceUi.Error) return
    // **ولا لوحةَ بلا بديل** — البند ٥.
    if (choices.alternatives.isEmpty()) return

    val preview = ui.previewRouteId

    Column(
        modifier = modifier
            .fillMaxWidth()
            .padding(horizontal = RahalSpaceTokens.md, vertical = RahalSpaceTokens.sm)
            .clip(RahalShapeTokens.md)
            .background(Rahal.colors.surface)
            .padding(RahalSpaceTokens.sm),
        verticalArrangement = Arrangement.spacedBy(RahalSpaceTokens.xs),
    ) {
        RouteCard(
            option = choices.recommended,
            title = stringResource(R.string.route_recommended),
            accent = MapRoutePalette.ACTIVE,
            selected = preview == null,
            deltas = emptyList(),
            enabled = !ui.busy,
            onClick = { onSelect(choices.recommended.routeId) },
        )

        for (alt in choices.alternatives) {
            RouteCard(
                option = alt,
                title = stringResource(R.string.route_alternative),
                accent = if (alt.routeId == preview) {
                    MapRoutePalette.PREVIEW
                } else {
                    MapRoutePalette.ALTERNATIVE
                },
                selected = alt.routeId == preview,
                deltas = RouteMetricText.deltasOf(alt),
                enabled = !ui.busy,
                onClick = { onSelect(alt.routeId) },
            )
        }

        // ── وزرُّ الاعتماد لا يظهر إلّا في معاينة ────────────────
        if (preview != null) {
            Row(
                Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.spacedBy(RahalSpaceTokens.sm),
            ) {
                PanelButton(
                    label = stringResource(R.string.act_cancel),
                    accent = Rahal.colors.inkMuted,
                    enabled = !ui.busy,
                    onClick = onCancel,
                    modifier = Modifier.weight(1f),
                )
                PanelButton(
                    label = stringResource(R.string.route_confirm),
                    accent = Rahal.colors.brand,
                    enabled = !ui.busy,
                    onClick = onConfirm,
                    modifier = Modifier.weight(1f),
                )
            }
        }

        if (ui is RouteChoiceUi.Stale) {
            Text(
                text = stringResource(R.string.route_no_longer_available),
                color = Rahal.colors.inkMuted,
                fontSize = 13.sp,
                modifier = Modifier.padding(top = RahalSpaceTokens.xs),
            )
        }
    }
}

/**
 * **بطاقةُ خيارٍ واحد.**
 *
 * **وهدفُ اللمس ٤٨dp** (البند ١٠) **ولو كان الشكلُ المرئيُّ أصغر.**
 *
 * **والاختيارُ لا يُعرف باللون وحدَه** (البند ٢٩): **حدٌّ ظاهرٌ
 * وسِمةُ `selected` في الدلالات**، فقارئُ الشاشة يقرؤها.
 */
@Composable
private fun RouteCard(
    option: RouteOption,
    title: String,
    accent: String,
    selected: Boolean,
    deltas: List<RouteMetricText.Delta>,
    enabled: Boolean,
    onClick: () -> Unit,
) {
    val metrics = RouteMetricText.metricsOf(option)
    val color = Color(android.graphics.Color.parseColor(accent))
    /**
     * **النصوصُ تُقرأ من الموارد أوّلاً ثمّ تُركَّب.**
     *
     * **و`stringResource` دالّةٌ مركَّبة** — فلا تُنادى داخلَ
     * `joinToString` ولا داخلَ `semantics`. **فتُقرأ في قائمةٍ ثمّ
     * تُوصَل.**
     */
    val deltaParts = deltas.map { d ->
        stringResource(
            when (d.kind) {
                RouteMetricText.Delta.Kind.SHORTER -> R.string.route_delta_shorter
                RouteMetricText.Delta.Kind.LONGER -> R.string.route_delta_longer
                RouteMetricText.Delta.Kind.FASTER -> R.string.route_delta_faster
                RouteMetricText.Delta.Kind.SLOWER -> R.string.route_delta_slower
            },
            d.magnitude,
        )
    }
    val deltaLine = deltaParts.joinToString(" · ")
    val metricsLine = stringResource(R.string.route_metrics, metrics.duration, metrics.distance)
    val description = if (deltaParts.isEmpty()) {
        stringResource(R.string.route_a11y_metrics, title, metrics.duration, metrics.distance)
    } else {
        stringResource(R.string.route_a11y_delta, title, deltaParts.joinToString("، "))
    }

    Row(
        modifier = Modifier
            .fillMaxWidth()
            // **البند ١٠** — أدنى هدفِ لمسٍ يوصي به أندرويد.
            .defaultMinSize(minHeight = 48.dp)
            .clip(RahalShapeTokens.sm)
            .border(
                width = if (selected) 2.dp else 1.dp,
                color = if (selected) color else Rahal.colors.line,
                shape = RahalShapeTokens.sm,
            )
            .clickable(enabled = enabled, onClick = onClick)
            .padding(horizontal = RahalSpaceTokens.md, vertical = RahalSpaceTokens.sm)
            .clearAndSetSemantics {
                contentDescription = description
                this.selected = selected
            },
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.SpaceBetween,
    ) {
        Column(verticalArrangement = Arrangement.spacedBy(2.dp)) {
            Text(
                text = title,
                color = color,
                fontSize = 14.sp,
                fontWeight = if (selected) FontWeight.Bold else FontWeight.Medium,
            )
            if (deltas.isEmpty()) {
                Text(text = metricsLine, color = Rahal.colors.inkMuted, fontSize = 13.sp)
            } else {
                Text(text = deltaLine, color = Rahal.colors.ink, fontSize = 13.sp)
            }
        }
    }
}

@Composable
private fun PanelButton(
    label: String,
    accent: Color,
    enabled: Boolean,
    onClick: () -> Unit,
    modifier: Modifier = Modifier,
) {
    Row(
        modifier = modifier
            .defaultMinSize(minHeight = 48.dp)
            .clip(RahalShapeTokens.pill)
            .background(if (enabled) accent else Rahal.colors.line)
            .clickable(enabled = enabled, onClick = onClick)
            .padding(horizontal = RahalSpaceTokens.lg, vertical = RahalSpaceTokens.sm),
        horizontalArrangement = Arrangement.Center,
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Text(text = label, color = Rahal.colors.onBrand, fontSize = 15.sp)
    }
}

