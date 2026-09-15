package com.rahalgo.merchant.offers

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.rahalgo.design.Rahal
import com.rahalgo.merchant.R
import com.rahalgo.ui.OfferCard
import com.rahalgo.ui.OfferDuration
import com.rahalgo.ui.RahalButton

/**
 * ══════════════════════════════════════════════════════════════════════
 * **شاشةُ العروض — ما هو سارٍ وما ينتظر** (`MO`، ٢٠٢٦-٠٩-١٥)
 * ══════════════════════════════════════════════════════════════════════
 *
 * # ولا معرّفاتٌ على شاشة
 *
 * **واسمُ الصنف هو ما يعرفه صاحبُ المتجر** — **ومعرّفٌ سداسيٌّ عشريٌّ
 * لا يقول شيئاً.**
 *
 * # والحالُ بلفظها
 *
 * **و«ساري» و«مجدول» و«موقوف» و«منتهي»** — **لا `scheduled`.**
 *
 * # وزرُّ الإيقاف حيث يُفيد
 *
 * **ولا يُعرَض على منتهٍ** — **وزرٌّ يُضغط ولا يقع شيءٌ يُقرأ معطوباً.**
 */
@Composable
fun OffersScreen(vm: OffersViewModel) {
    val ctx = LocalContext.current
    var itemId by remember { mutableStateOf("") }
    var percent by remember { mutableStateOf("") }
    var hours by remember { mutableStateOf(24) }

    LazyColumn(
        Modifier.fillMaxWidth().padding(14.dp),
        verticalArrangement = Arrangement.spacedBy(10.dp),
    ) {
        item {
            Text(
                stringResource(R.string.offers_title),
                style = MaterialTheme.typography.titleMedium,
                fontWeight = FontWeight.Bold,
            )
        }

        if (vm.error.isNotEmpty()) {
            item {
                Text(
                    vm.error,
                    color = Rahal.colors.danger,
                    modifier = Modifier
                        .fillMaxWidth()
                        .clip(Rahal.shape.md)
                        .background(Rahal.colors.warnTint)
                        .padding(12.dp),
                )
            }
        }

        // ══════════════════════════════════════════════════════════════
        // **وإنشاءُ عرضٍ في أعلى الشاشة** (`MO-02`)
        // ══════════════════════════════════════════════════════════════
        //
        // **ومن فتح «العروض» أكثرَ ما يريده أن يُنزل عرضاً** — **وزرٌّ
        // في آخر قائمةٍ طويلةٍ لا يُبلَغ.**
        item {
            Column(
                Modifier
                    .fillMaxWidth()
                    .clip(Rahal.shape.md)
                    .background(Rahal.colors.canvas)
                    .padding(12.dp),
            ) {
                Text(stringResource(R.string.offer_new), fontWeight = FontWeight.Bold)
                Spacer(Modifier.height(8.dp))

                // **والصنفُ يُختار من قائمته** — **ولا يُكتب معرّفُه.**
                Text(
                    stringResource(R.string.offer_pick_item),
                    style = MaterialTheme.typography.bodySmall,
                    color = Rahal.colors.inkMuted,
                )
                Spacer(Modifier.height(4.dp))
                vm.items.forEach { it0 ->
                    val chosen = it0.id == itemId
                    Text(
                        text = it0.name,
                        maxLines = 1,
                        color = if (chosen) Rahal.colors.accent else Rahal.colors.ink,
                        fontWeight = if (chosen) FontWeight.Bold else FontWeight.Normal,
                        modifier = Modifier
                            .fillMaxWidth()
                            .clickable { itemId = it0.id }
                            .padding(vertical = 6.dp),
                    )
                }

                Spacer(Modifier.height(8.dp))
                OutlinedTextField(
                    value = percent,
                    onValueChange = { v -> percent = v.filter { it.isDigit() }.take(2) },
                    label = { Text(stringResource(R.string.offer_percent)) },
                    singleLine = true,
                    modifier = Modifier.fillMaxWidth(),
                )

                Spacer(Modifier.height(8.dp))
                Text(
                    stringResource(R.string.offer_duration),
                    style = MaterialTheme.typography.bodySmall,
                    color = Rahal.colors.inkMuted,
                )
                Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.spacedBy(6.dp)) {
                    // **ومددٌ تُضغط ضغطةً** — **ولا يُكتب تاريخٌ بالأصابع.**
                    OfferDuration.PRESET_HOURS.take(5).forEach { h ->
                        Text(
                            text = OfferDuration.label(ctx, h),
                            style = MaterialTheme.typography.bodySmall,
                            color = if (h == hours) Rahal.colors.accent else Rahal.colors.inkMuted,
                            fontWeight = if (h == hours) FontWeight.Bold else FontWeight.Normal,
                            modifier = Modifier.clickable { hours = h }.padding(6.dp),
                        )
                    }
                }

                Spacer(Modifier.height(6.dp))
                // **ومن يتحمّله مكتوبٌ قبل الضغط** — **ولا يُكتشَف في
                // كشف الحساب آخرَ الشهر.**
                Text(
                    stringResource(R.string.offer_borne_note),
                    style = MaterialTheme.typography.bodySmall,
                    color = Rahal.colors.inkMuted,
                )

                Spacer(Modifier.height(8.dp))
                RahalButton(
                    onClick = { vm.create(itemId, percent.toIntOrNull() ?: 0, hours, "") },
                    enabled = !vm.busy,
                    modifier = Modifier.fillMaxWidth(),
                ) {
                    Text(stringResource(R.string.offer_create))
                }
            }
        }

        val rows = vm.rows
        if (rows != null && rows.isEmpty()) {
            item {
                Text(
                    stringResource(R.string.offers_empty),
                    color = Rahal.colors.inkMuted,
                    modifier = Modifier.padding(vertical = 12.dp),
                )
            }
        }
        items(rows.orEmpty(), key = { it.id }) { o ->
            // **والبطاقةُ مركزيّةٌ** — **يقرؤها المندوبُ كما يقرؤها هو.**
            OfferCard(
                itemName = o.itemName,
                status = o.status,
                priceBefore = o.priceBefore,
                priceAfter = o.priceAfter,
                percent = o.discountPercent,
                stopping = vm.stopping == o.id,
                onStop = { vm.stop(o.id) },
            )
        }

        if (vm.busy) {
            item {
                Box(Modifier.fillMaxWidth().padding(16.dp), Alignment.Center) {
                    CircularProgressIndicator(color = Rahal.colors.accent)
                }
            }
        }
    }
}
