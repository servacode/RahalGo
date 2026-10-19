package com.rahalgo.merchant.menu

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.unit.dp
import com.rahalgo.design.Rahal
import com.rahalgo.merchant.R
import com.rahalgo.ui.RahalButton
import com.rahalgo.ui.RahalOutlineButton
import com.rahalgo.ui.RahalTextButton
import com.rahalgo.ui.Screen
import com.rahalgo.ui.ScreenTitle

/**
 * **زرُّ «قائمة المشروبات»** فوق شبكة الأقسام (قرارُ المالك ٢٠٢٦-١٠-١٩).
 */
@Composable
internal fun DrinksButton(vm: MenuViewModel) {
    Spacer(Modifier.height(6.dp))
    RahalOutlineButton(onClick = { vm.openDrinks() }, modifier = Modifier.fillMaxWidth()) {
        Text(stringResource(R.string.drinks_title))
    }
}

/**
 * ══════════════════════════════════════════════════════════════════════
 * **قائمةُ المشروبات — تُكتب مرّةً وتنضاف لكلّ الأصناف** (قرارُ المالك ٢٠٢٦-١٠-١٩)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **والمحرّكُ ينسخها** إلى كلّ صنفٍ مجموعةَ «مشروبات» اختياريّة، **ويتبع الدولاريُّ
 * سعرَ الصرف** كالأصناف. **ومرآتُها `DrinksCard.tsx` في الويب.**
 */
@Composable
internal fun DrinksEditor(vm: MenuViewModel) {
    Screen {
        ScreenTitle(stringResource(R.string.drinks_title), stringResource(R.string.drinks_hint))
        Row(
            Modifier.clip(Rahal.shape.md).background(Rahal.colors.canvas),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            listOf(false to R.string.drinks_syp, true to R.string.drinks_usd).forEach { (usd, label) ->
                val on = vm.drinksUsd == usd
                Text(
                    stringResource(label),
                    color = if (on) Rahal.colors.canvas else Rahal.colors.ink,
                    modifier = Modifier
                        .background(if (on) Rahal.colors.brand else Rahal.colors.canvas)
                        .clickable { vm.drinksUsd = usd }
                        .padding(horizontal = 16.dp, vertical = 8.dp),
                )
            }
        }
        Spacer(Modifier.height(8.dp))
        if (vm.drinks.isEmpty()) {
            Text(stringResource(R.string.drinks_empty), color = Rahal.colors.inkMuted)
        }
        vm.drinks.forEachIndexed { i, r ->
            Row(verticalAlignment = Alignment.CenterVertically) {
                OutlinedTextField(
                    value = r.name,
                    onValueChange = { vm.editDrink(i, r.copy(name = it)) },
                    label = { Text(stringResource(R.string.drinks_name)) },
                    singleLine = true,
                    modifier = Modifier.weight(1f),
                )
                Spacer(Modifier.width(6.dp))
                OutlinedTextField(
                    value = if (vm.drinksUsd) r.usd else r.syp,
                    onValueChange = { v ->
                        if (vm.drinksUsd) {
                            vm.editDrink(i, r.copy(usd = v.filter { c -> c.isDigit() || c == '.' || c == '٫' || c == ',' }.take(8)))
                        } else {
                            vm.editDrink(i, r.copy(syp = v.filter { it.isDigit() }.take(9)))
                        }
                    },
                    label = { Text(stringResource(if (vm.drinksUsd) R.string.drinks_price_usd else R.string.drinks_price_syp)) },
                    singleLine = true,
                    keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Number),
                    modifier = Modifier.width(110.dp),
                )
                RahalTextButton(onClick = { vm.removeDrink(i) }) { Text("×", color = Rahal.colors.danger) }
            }
        }
        Spacer(Modifier.height(6.dp))
        RahalOutlineButton(onClick = { vm.addDrink() }) { Text(stringResource(R.string.drinks_add)) }
        if (vm.drinksError.isNotEmpty()) {
            Spacer(Modifier.height(6.dp))
            Text(vm.drinksError, color = Rahal.colors.danger, style = MaterialTheme.typography.bodySmall)
        }
        Spacer(Modifier.height(12.dp))
        Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
            RahalButton(onClick = { vm.saveDrinks() }, enabled = !vm.busy) { Text(stringResource(R.string.drinks_save)) }
            RahalTextButton(onClick = { vm.closeDrinks() }) { Text(stringResource(R.string.act_cancel)) }
        }
    }
}
