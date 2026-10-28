package com.rahalgo.ui.menu

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.selection.selectable
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.semantics.Role
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.unit.dp
import com.rahalgo.design.Rahal
import com.rahalgo.shared.merchant.Drink
import com.rahalgo.ui.AppCore
import com.rahalgo.ui.Flash
import com.rahalgo.ui.R
import com.rahalgo.ui.RahalButton
import com.rahalgo.ui.RahalOutlineButton
import com.rahalgo.ui.Screen
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.launch

/**
 * ══════════════════════════════════════════════════════════════════════
 * **قائمةُ المشروبات — تُكتب مرّةً وتنضاف لكلّ الأصناف** (قرارُ المالك ٢٠٢٦-١٠-١٩)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **والمحرّكُ ينسخها** إلى كلّ صنفٍ مجموعةَ «مشروبات» اختياريّة، **ويتبع الدولاريُّ
 * سعرَ الصرف** كالأصناف. **ومرآتُها `DrinksCard.tsx` في الويب.**
 *
 * **ومشتركةٌ بين المتجر والمندوب** (طلبُ المالك ٢٠٢٦-١٠-١٩: «تطبيق المندوب لازم يكون
 * مطابق للمتجر») — **والفرقُ بينهما بابُ الشبكة وحدَه** (`load`/`save`).
 */
class DrinksState(
    private val scope: CoroutineScope,
    private val load: suspend () -> List<Drink>,
    private val save: suspend (List<Drink>) -> List<Drink>,
    /** **كم صنفاً في القائمة** — لرسالة «انضافت لـN صنف». */
    private val itemCount: () -> Int,
    private val errText: (Throwable) -> String,
) {
    /** **سطرٌ في المحرّر** — النصُّ كما يُكتب، والمحرّكُ يحسب. */
    data class Row(val name: String = "", val syp: String = "", val usd: String = "")

    var rows by mutableStateOf<List<Row>>(emptyList())
        private set
    var usd by mutableStateOf(false)
    var open by mutableStateOf(false)
        private set
    var error by mutableStateOf("")
        private set
    var busy by mutableStateOf(false)
        private set

    private fun fill(list: List<Drink>) {
        usd = list.any { it.priceUsd != null }
        rows = list.map { d ->
            Row(
                name = d.name,
                syp = d.price.toString(),
                usd = d.priceUsd?.let { v -> if (v == Math.floor(v)) v.toLong().toString() else v.toString() }.orEmpty(),
            )
        }
    }

    fun show() {
        error = ""
        open = true
        scope.launch {
            runCatching { load() }
                .onSuccess { fill(it) }
                .onFailure { error = errText(it) }
        }
    }

    fun close() {
        open = false
    }

    fun edit(i: Int, row: Row) {
        error = ""
        rows = rows.mapIndexed { j, r -> if (j == i) row else r }
    }

    fun add() {
        rows = rows + Row()
    }

    fun remove(i: Int) {
        rows = rows.filterIndexed { j, _ -> j != i }
    }

    fun submit() {
        val app = AppCore.get().app
        if (rows.any { it.name.isBlank() }) {
            error = app.getString(R.string.drinks_v_name)
            return
        }
        busy = true
        scope.launch {
            val body = rows.map { r ->
                if (usd) {
                    Drink(name = r.name.trim(), priceUsd = r.usd.replace('٫', '.').replace(',', '.').toDoubleOrNull() ?: 0.0)
                } else {
                    Drink(name = r.name.trim(), price = r.syp.toLongOrNull() ?: 0L)
                }
            }
            runCatching { save(body) }
                .onSuccess {
                    fill(it)
                    open = false
                    // **ويُقال له ما صار** (طلبُ المالك ٢٠٢٦-١٠-١٩): كم صنفاً نالها.
                    val n = itemCount()
                    Flash.ok(
                        when {
                            it.isEmpty() -> app.getString(R.string.drinks_cleared)
                            n == 0 -> app.getString(R.string.drinks_saved_none)
                            else -> app.getString(R.string.drinks_saved, n)
                        },
                    )
                }
                .onFailure { error = errText(it) }
            busy = false
        }
    }
}

/**
 * **زرُّ «قائمة المشروبات»** فوق شبكة الأقسام — **ممتلئٌ بلون المنصّة وأيقونةِ «+»**
 * (طلبُ المالك ٢٠٢٦-١٠-١٩: «سمّيها واعطيها لون واضح مع ايقونة اضافة لتبين»).
 */
@Composable
fun DrinksButton(state: DrinksState) {
    Spacer(Modifier.height(6.dp))
    RahalButton(onClick = { state.show() }, modifier = Modifier.fillMaxWidth()) {
        Icon(
            painter = painterResource(R.drawable.ic_plus),
            contentDescription = null,
            modifier = Modifier.size(18.dp),
        )
        Spacer(Modifier.width(8.dp))
        Text(stringResource(R.string.drinks_button), fontWeight = FontWeight.Bold)
    }
}

@Composable
fun DrinksEditor(
    state: DrinksState,
    // **وسطرُ الشرح أُلغي** (ملاحظةُ المالك ٢٠٢٦-١٠-٢٩: «ما في داعي للشرح») —
    // ويبقى الوسيطُ فلا ينكسر نداءُ المندوب.
    @Suppress("UNUSED_PARAMETER") hint: Int = R.string.drinks_hint,
) {
    Screen {
        ToolHeader(stringResource(R.string.drinks_title)) { state.close() }
        // ── العملة: «ليرة / دولار» زرّان متساويان ──
        Row(
            Modifier
                .fillMaxWidth()
                .height(RowFieldHeight)
                .clip(Rahal.shape.md)
                .border(Rahal.stroke.hair, Rahal.colors.line, Rahal.shape.md),
        ) {
            listOf(false to R.string.drinks_syp, true to R.string.drinks_usd).forEach { (usd, label) ->
                val on = state.usd == usd
                Box(
                    Modifier
                        .weight(1f)
                        .height(RowFieldHeight)
                        .background(if (on) Rahal.colors.brand else Color.Transparent)
                        .selectable(selected = on, enabled = !state.busy, role = Role.RadioButton) { state.usd = usd },
                    contentAlignment = Alignment.Center,
                ) {
                    Text(
                        stringResource(label),
                        color = if (on) Rahal.colors.onBrand else Rahal.colors.ink,
                        fontWeight = FontWeight.Bold,
                    )
                }
            }
        }
        Spacer(Modifier.height(Rahal.space.md))
        if (state.rows.isEmpty()) {
            Text(stringResource(R.string.drinks_empty), color = Rahal.colors.inkMuted)
        } else {
            // **عناوينُ الأعمدة مرّةً فوق الأسطر** — كسطور خيارات الصنف.
            Row(verticalAlignment = Alignment.CenterVertically) {
                ColumnLabel(stringResource(R.string.drinks_name), Modifier.weight(1f))
                Spacer(Modifier.width(Rahal.space.xs))
                ColumnLabel(
                    stringResource(if (state.usd) R.string.drinks_price_usd else R.string.drinks_price_syp),
                    Modifier.width(DrinkPriceWidth),
                )
                Spacer(Modifier.width(TouchTarget + Rahal.space.xs))
            }
        }
        state.rows.forEachIndexed { i, r ->
            Row(
                Modifier
                    .fillMaxWidth()
                    .padding(vertical = 2.dp),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                RowField(
                    value = r.name,
                    onChange = { state.edit(i, r.copy(name = it)) },
                    placeholder = stringResource(R.string.drinks_name),
                    enabled = !state.busy,
                    modifier = Modifier.weight(1f),
                )
                Spacer(Modifier.width(Rahal.space.xs))
                RowField(
                    value = if (state.usd) r.usd else r.syp,
                    onChange = { v ->
                        if (state.usd) {
                            state.edit(i, r.copy(usd = v.filter { c -> c.isDigit() || c == '.' || c == '٫' || c == ',' }.take(8)))
                        } else {
                            state.edit(i, r.copy(syp = v.filter { it.isDigit() }.take(9)))
                        }
                    },
                    placeholder = "0",
                    keyboard = if (state.usd) KeyboardType.Decimal else KeyboardType.Number,
                    enabled = !state.busy,
                    modifier = Modifier.width(DrinkPriceWidth),
                )
                Spacer(Modifier.width(Rahal.space.xs))
                IconButton(onClick = { state.remove(i) }, enabled = !state.busy, modifier = Modifier.size(TouchTarget)) {
                    Icon(
                        painter = painterResource(R.drawable.ic_trash),
                        contentDescription = stringResource(R.string.drinks_delete),
                        tint = Rahal.colors.danger,
                    )
                }
            }
        }
        Spacer(Modifier.height(Rahal.space.sm))
        RahalOutlineButton(onClick = { state.add() }, enabled = !state.busy) {
            IconText(R.drawable.ic_plus, stringResource(R.string.drinks_add))
        }
        ToolError(state.error)
        Spacer(Modifier.height(Rahal.space.lg))
        ToolActions(
            primary = stringResource(R.string.act_save),
            primaryEnabled = !state.busy,
            onPrimary = { state.submit() },
            cancelEnabled = !state.busy,
            onCancel = { state.close() },
        )
    }
}

/** **عرضُ حقل سعر المشروب** — كحقل سعر الخيار. */
private val DrinkPriceWidth = 96.dp
