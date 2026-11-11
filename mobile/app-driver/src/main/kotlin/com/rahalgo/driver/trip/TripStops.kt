package com.rahalgo.driver.trip

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextDecoration
import com.rahalgo.design.Rahal
import com.rahalgo.driver.R
import com.rahalgo.shared.model.DriverOrder
import com.rahalgo.shared.model.PickupStop
import com.rahalgo.ui.InvoiceLines
import com.rahalgo.ui.RahalOutlineButton
import com.rahalgo.ui.money

/**
 * ══════════════════════════════════════════════════════════════════════
 * **كتلةُ المحطّات** — طلبٌ من عدّة متاجر (٢٠٢٦-١١-٠٦)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **لكلّ محطّةٍ مربّع**: الاسمُ وحالُها وما تدفعه وأصنافُها (مطويّة) وزرُّها. **والمنشولةُ مشطوبة.**
 */
@Composable
internal fun StopsBlock(
    order: DriverOrder,
    currentId: String?,
    busy: Boolean,
    onStep: (to: String, stop: String) -> Unit,
) {
    val stops = order.stops ?: return
    Column(Modifier.fillMaxWidth(), verticalArrangement = Arrangement.spacedBy(Rahal.space.sm)) {
        Text(
            stringResource(R.string.stops_title),
            color = Rahal.colors.inkMuted,
            fontWeight = FontWeight.Bold,
            style = Rahal.type.caption,
        )
        stops.sortedBy { it.seq }.forEach { s ->
            StopCard(order, s, s.merchantId == currentId, busy, onStep)
        }
    }
}

/** **نصُّ زرّ المحطّة** — «وصلت X» · «استلمت من X» · «استلم بضاعة X من السائق السابق». */
@Composable
internal fun stopButtonText(s: PickupStop, to: String): String = when {
    to == "at_pickup" -> stringResource(R.string.stop_arrive_named, s.name)
    s.handover -> stringResource(R.string.stop_handover_named, s.name)
    else -> stringResource(R.string.stop_pick_named, s.name)
}

@Composable
private fun StopCard(
    order: DriverOrder,
    s: PickupStop,
    current: Boolean,
    busy: Boolean,
    onStep: (String, String) -> Unit,
) {
    val removed = !stopLive(s)
    val tint = when {
        removed -> Rahal.colors.inkMuted
        stopDone(s) -> Rahal.colors.success
        current -> Rahal.colors.brand
        else -> Rahal.colors.inkMuted
    }
    var open by remember(s.merchantId) { mutableStateOf(false) }
    Column(
        Modifier
            .fillMaxWidth()
            .clip(Rahal.shape.md)
            .background(tint.copy(alpha = 0.10f))
            .padding(horizontal = Rahal.space.md, vertical = Rahal.space.md),
        verticalArrangement = Arrangement.spacedBy(Rahal.space.sm),
    ) {
        Row(
            Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.SpaceBetween,
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Text(
                s.name,
                color = if (removed) Rahal.colors.inkMuted else Rahal.colors.ink,
                fontWeight = FontWeight.Bold,
                textDecoration = if (removed) TextDecoration.LineThrough else null,
                style = Rahal.type.body,
                modifier = Modifier.weight(1f),
            )
            Box(
                Modifier
                    .clip(Rahal.shape.pill)
                    .background(tint.copy(alpha = 0.18f))
                    .padding(horizontal = Rahal.space.md, vertical = Rahal.space.xs),
            ) {
                Text(
                    stringResource(
                        when {
                            removed -> R.string.stop_removed
                            s.mine -> R.string.stop_chip_done
                            s.handover -> R.string.stop_chip_handover
                            s.arrivedAt != null -> R.string.stop_chip_arrived
                            current -> R.string.stop_chip_next
                            else -> R.string.stop_chip_waiting
                        },
                    ),
                    color = tint,
                    fontWeight = FontWeight.Bold,
                    style = Rahal.type.caption,
                )
            }
        }
        if (removed) return@Column
        if (s.address.isNotBlank()) {
            Text(s.address, color = Rahal.colors.inkMuted, style = Rahal.type.caption)
        }
        // **ما يدفعه لهذه المحطّة** — أو «مدفوع بالمحفظة». وما استُلمت لا يُدفع لها ثانيةً.
        if (!s.mine) {
            if (s.paysBy == "wallet") {
                Text(
                    stringResource(R.string.stop_pay_wallet),
                    color = Rahal.colors.success,
                    fontWeight = FontWeight.Bold,
                    style = Rahal.type.body,
                )
            } else if (s.payAmount > 0) {
                Text(
                    stringResource(R.string.stop_pay_cash, money(s.payAmount)),
                    color = Rahal.colors.brand,
                    fontWeight = FontWeight.Bold,
                    style = Rahal.type.body,
                )
            }
        }
        // **الأصناف مطويّةٌ** — لمسةٌ تفتحها.
        if (s.items.isNotEmpty()) {
            Text(
                stringResource(R.string.trip_order_items) + " (" + s.items.sumOf { it.qty } + ")",
                color = Rahal.colors.brand,
                fontWeight = FontWeight.Bold,
                style = Rahal.type.caption,
                modifier = Modifier.clickable { open = !open },
            )
            if (open) InvoiceLines(driverInvoiceLines(order.copy(items = s.items)))
        }
        stopAction(s)?.let { a ->
            RahalOutlineButton(
                onClick = { onStep(a.status, s.merchantId) },
                enabled = !busy,
                modifier = Modifier.fillMaxWidth(),
            ) {
                Text(stopButtonText(s, a.status), fontWeight = FontWeight.Bold)
            }
        }
    }
}
