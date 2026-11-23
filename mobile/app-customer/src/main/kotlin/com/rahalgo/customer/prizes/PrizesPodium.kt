package com.rahalgo.customer.prizes

import androidx.compose.foundation.Image
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.draw.shadow
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.dp
import com.rahalgo.customer.R
import com.rahalgo.design.Rahal
import com.rahalgo.ui.RahalCard
import com.rahalgo.ui.money

/**
 * ══════════════════════════════════════════════════════════════════════
 * **منصّةُ التتويج — خمسةُ مربّعات، الأوّلُ أعلاها في الوسط وعليه كأس** (ملاحظةُ المالك
 * ٢٠٢٦-١١-٢٤: «مثل كاس العالم… بس بدل ٣، ٥ فائزين»)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **الترتيبُ من اليمين** (الواجهةُ عربيّة): الرابع، الثاني، **الأوّل**، الثالث، الخامس —
 * **فالثاني جنبَ الأوّل حيث تبدأ العين.** وكلُّ مربّعٍ بارتفاعه وبلونٍ لمرتبته.
 */
@Composable
fun Podium(amounts: List<Long>) {
    if (amounts.isEmpty()) return
    val ranks = PODIUM_ORDER.filter { it <= amounts.size }
    RahalCard(modifier = Modifier.fillMaxWidth()) {
        Text(
            stringResource(R.string.prizes_ladder_title),
            color = Rahal.colors.ink,
            style = Rahal.type.sectionTitle,
            modifier = Modifier.fillMaxWidth(),
            textAlign = TextAlign.Center,
        )
        Spacer(Modifier.height(Rahal.space.md))
        Row(
            Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.spacedBy(4.dp),
            verticalAlignment = Alignment.Bottom,
        ) {
            for (rank in ranks) {
                Step(rank, amounts[rank - 1], Modifier.weight(if (rank == 1) 1.25f else 1f))
            }
        }
    }
}

@Composable
private fun Step(rank: Int, amount: Long, modifier: Modifier) {
    val (top, bottom) = STEP_COLORS[rank - 1]
    Column(modifier, horizontalAlignment = Alignment.CenterHorizontally) {
        if (rank == 1) {
            Image(
                painter = painterResource(R.drawable.trophy_3d),
                contentDescription = null,
                modifier = Modifier.size(64.dp),
            )
        }
        Text(
            money(amount),
            color = Rahal.colors.ink,
            style = if (rank == 1) Rahal.type.sectionTitle else Rahal.type.label,
            fontWeight = FontWeight.Bold,
            textAlign = TextAlign.Center,
            maxLines = 1,
        )
        Spacer(Modifier.height(Rahal.space.xxs))
        Box(
            Modifier
                .fillMaxWidth()
                .height(STEP_HEIGHTS[rank - 1])
                .shadow(3.dp, Rahal.shape.md)
                .clip(Rahal.shape.md)
                .background(Brush.verticalGradient(listOf(top, bottom))),
            contentAlignment = Alignment.TopCenter,
        ) {
            Column(horizontalAlignment = Alignment.CenterHorizontally) {
                Spacer(Modifier.height(Rahal.space.xs))
                Text(
                    rank.toString(),
                    color = Color.White,
                    style = if (rank == 1) Rahal.type.headline else Rahal.type.sectionTitle,
                    fontWeight = FontWeight.Black,
                )
                Text(
                    stringResource(PLACE_LABELS[rank - 1]),
                    color = Color.White,
                    style = Rahal.type.badge,
                    textAlign = TextAlign.Center,
                    maxLines = 2,
                    modifier = Modifier.padding(horizontal = 2.dp),
                )
            }
        }
    }
}

/**
 * **شروطُ المسابقة بتصميمٍ واضح** (ملاحظةُ المالك ٢٠٢٦-١١-٢٤) — بنودٌ مرقّمةٌ بدوائر،
 * **وأرقامُها من المحرّك** (عددُ الدعوات وعددُ الرابحين) فلا يُكتب رقمٌ ويتبدّل في اللوحة.
 */
@Composable
fun PrizeRules(perEntry: Int, winners: Int) {
    val rules = listOf(
        stringResource(R.string.prizes_rule_invite, perEntry),
        stringResource(R.string.prizes_rule_more, perEntry),
        stringResource(R.string.prizes_rule_new),
        stringResource(R.string.prizes_rule_draw),
        stringResource(R.string.prizes_rule_winners, winners),
        stringResource(R.string.prizes_rule_use),
        stringResource(R.string.prizes_rule_diff),
        stringResource(R.string.prizes_rule_expire),
    )
    RahalCard(modifier = Modifier.fillMaxWidth()) {
        Text(stringResource(R.string.prizes_rules_title), color = Rahal.colors.ink, style = Rahal.type.sectionTitle)
        Spacer(Modifier.height(Rahal.space.sm))
        rules.forEachIndexed { i, line ->
            Row(
                Modifier
                    .fillMaxWidth()
                    .padding(vertical = Rahal.space.xs),
                verticalAlignment = Alignment.Top,
            ) {
                Box(
                    Modifier
                        .size(26.dp)
                        .clip(CircleShape)
                        .background(Rahal.colors.brand),
                    contentAlignment = Alignment.Center,
                ) {
                    Text((i + 1).toString(), color = Rahal.colors.onBrand, style = Rahal.type.badge, fontWeight = FontWeight.Bold)
                }
                Spacer(Modifier.width(Rahal.space.sm))
                Text(line, color = Rahal.colors.ink, style = Rahal.type.body, modifier = Modifier.weight(1f))
            }
        }
    }
}

/** **من اليمين**: الرابع، الثاني، الأوّل، الثالث، الخامس. */
private val PODIUM_ORDER = listOf(4, 2, 1, 3, 5)

private val STEP_HEIGHTS: List<Dp> = listOf(118.dp, 94.dp, 80.dp, 64.dp, 56.dp)

/** **ذهبٌ وفضّةٌ وبرونزٌ ثمّ لونان هادئان** — ألوانُ المرتبة لا ألوانُ الثيم، فهي هي في الوضعين. */
private val STEP_COLORS = listOf(
    Color(0xFFFFC531) to Color(0xFFD98C00),
    Color(0xFFC9D1DB) to Color(0xFF8A95A3),
    Color(0xFFE3A06A) to Color(0xFFA9622B),
    Color(0xFF7FB8E6) to Color(0xFF3F7FB5),
    Color(0xFF9FD3B0) to Color(0xFF4E9A6C),
)

private val PLACE_LABELS = listOf(
    R.string.prizes_place_1,
    R.string.prizes_place_2,
    R.string.prizes_place_3,
    R.string.prizes_place_4,
    R.string.prizes_place_5,
)
