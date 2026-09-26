package com.rahalgo.merchant.store

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.background
import androidx.compose.material3.DropdownMenu
import androidx.compose.material3.DropdownMenuItem
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.rahalgo.design.Rahal
import com.rahalgo.merchant.R
import com.rahalgo.shared.merchant.Store

/**
 * ══════════════════════════════════════════════════════════════════════
 * **مبدّلُ الفرع — لمالكٍ يملك أكثرَ من متجر** (B8، ٢٠٢٦-٠٩-٢٦)
 * ══════════════════════════════════════════════════════════════════════
 *
 * (قرارُ المالك: «بسيطٌ إن كان واحداً، ومبدّلٌ إن كان أكثر».)
 *
 * # ولا يظهر لمن يملك واحداً
 *
 * **شريطٌ يقول اسمَ متجرٍ واحدٍ لا معنى له** — يشغل مكاناً ويوهم بخيارٍ لا
 * وجودَ له. **فيُخفى ما لم يكن هناك فرعان فأكثر**، وصاحبُ الفرع الواحد لا
 * يرى إلّا شاشتَه.
 *
 * # وتبديلُه يُعيد بناءَ الشاشات كلِّها
 *
 * **الاختيارُ مشتركٌ** (`SelectedStore`)، **والنبضةُ تُبثّ بعده**
 * (`Refresh.bump`) — فالطلباتُ والقائمةُ و«متجري» تُقرأ للفرع الجديد، لا
 * فرعٌ هنا وفرعٌ هناك.
 */
@Composable
fun StoreSwitcher(
    stores: List<Store>,
    selectedId: String,
    onSelect: (String) -> Unit,
) {
    if (stores.size <= 1) return
    var expanded by remember { mutableStateOf(false) }
    val current = stores.firstOrNull { it.id == selectedId } ?: stores.first()

    Row(
        Modifier
            .fillMaxWidth()
            .padding(horizontal = 16.dp, vertical = 6.dp)
            .background(Rahal.colors.field, RoundedCornerShape(10.dp))
            .clickable { expanded = true }
            .padding(horizontal = 12.dp, vertical = 10.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Icon(
            painterResource(com.rahalgo.ui.R.drawable.ic_store),
            contentDescription = null,
            modifier = Modifier.size(18.dp),
            tint = Rahal.colors.brand,
        )
        Text(
            current.name,
            fontWeight = FontWeight.Bold,
            style = MaterialTheme.typography.bodyMedium,
            modifier = Modifier
                .padding(start = 8.dp)
                .weight(1f),
        )
        Text(
            stringResource(R.string.store_switch),
            color = Rahal.colors.brand,
            style = MaterialTheme.typography.labelMedium,
        )

        DropdownMenu(expanded = expanded, onDismissRequest = { expanded = false }) {
            stores.forEach { s ->
                DropdownMenuItem(
                    text = {
                        Text(
                            s.name,
                            fontWeight = if (s.id == current.id) FontWeight.Bold else FontWeight.Normal,
                        )
                    },
                    onClick = {
                        expanded = false
                        if (s.id != current.id) onSelect(s.id)
                    },
                )
            }
        }
    }
}
