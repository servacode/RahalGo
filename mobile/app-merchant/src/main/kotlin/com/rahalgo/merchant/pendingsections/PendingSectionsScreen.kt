package com.rahalgo.merchant.pendingsections

import androidx.activity.compose.BackHandler
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material3.Checkbox
import androidx.compose.material3.CheckboxDefaults
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.ModalBottomSheet
import androidx.compose.material3.Text
import androidx.compose.material3.rememberModalBottomSheetState
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.style.TextOverflow
import com.rahalgo.design.Rahal
import com.rahalgo.merchant.R
import com.rahalgo.shared.merchant.PendingSyncItem
import com.rahalgo.shared.merchant.PlatformSectionRef
import com.rahalgo.ui.BottomActionBar
import com.rahalgo.ui.CardVariant
import com.rahalgo.ui.LabeledField
import com.rahalgo.ui.ListRow
import com.rahalgo.ui.RahalCard
import com.rahalgo.ui.SheetHeader
import com.rahalgo.ui.StatusTone
import com.rahalgo.ui.ConfirmDialog
import com.rahalgo.ui.Empty
import com.rahalgo.ui.LoadState
import com.rahalgo.ui.RahalButton
import com.rahalgo.ui.RahalOutlineButton
import com.rahalgo.ui.RahalTextButton
import com.rahalgo.ui.menu.priceLine

/** من يطلب ورقة اختيار القسم: صنف واحد، أو كل المحدّدين. */
private sealed interface PickFor {
    data class One(val itemId: String) : PickFor
    data object Selected : PickFor
}

/**
 * شاشة «أصناف بلا قسم» (قرار المالك ٢٠٢٦-١١-٠٧) — انظر [PendingSectionsState].
 * الأصناف 50 بالصفحة، وتتحمّل الصفحة التالية لما يوصل المستخدم آخر القائمة.
 */
@Composable
fun PendingSectionsScreen(state: PendingSectionsState) {
    BackHandler { if (state.selecting) state.stopSelecting() else state.close() }
    var pick by remember { mutableStateOf<PickFor?>(null) }
    val list = state.list

    Column(Modifier.fillMaxSize()) {
      Column(Modifier.weight(1f).fillMaxWidth().padding(horizontal = Rahal.space.gutter).padding(top = Rahal.space.lg)) {
        Row(Modifier.fillMaxWidth(), verticalAlignment = Alignment.CenterVertically) {
            Column(Modifier.weight(1f)) {
                Text(
                    stringResource(R.string.ps_title),
                    style = Rahal.type.screenTitle,
                    color = Rahal.colors.ink,
                )
                if (list.total > 0) {
                    Text(
                        stringResource(R.string.ps_waiting, list.total),
                        style = Rahal.type.caption,
                        color = Rahal.colors.brand,
                    )
                }
            }
            RahalTextButton(onClick = { state.close() }) { Text(stringResource(R.string.ps_close)) }
        }
        Spacer(Modifier.height(Rahal.space.sm))

        LazyColumn(
            Modifier.weight(1f).fillMaxWidth(),
            verticalArrangement = Arrangement.spacedBy(Rahal.space.sm),
        ) {
            item(key = "top") { TopPart(state) }
            when {
                state.loading && list.items.isEmpty() -> item(key = "load") { LoadState(true, "") }
                state.error.isNotEmpty() && list.items.isEmpty() ->
                    item(key = "err") { LoadState(false, state.error, onRetry = { state.reload() }) }
                list.items.isEmpty() -> item(key = "empty") {
                    Empty(stringResource(if (state.query.isBlank()) R.string.ps_empty else R.string.ps_empty_search))
                }
                else -> {
                    items(list.items, key = { it.id }) { item ->
                        ItemRow(state, item, onPick = { pick = PickFor.One(item.id) })
                    }
                    item(key = "more") { Footer(state) }
                }
            }
            if (list.total > 0 && !state.selecting) {
                item(key = "rest") {
                    RahalOutlineButton(
                        onClick = { state.askRest() },
                        enabled = !state.busy,
                        modifier = Modifier.fillMaxWidth(),
                    ) { Text(stringResource(R.string.ps_rest)) }
                    Spacer(Modifier.height(Rahal.space.lg))
                }
            }
        }
      }

        if (state.selecting && list.selectedCount > 0) {
            BottomActionBar(
                summary = {
                    Text(
                        stringResource(R.string.ps_selected, list.selectedCount),
                        style = Rahal.type.bodyStrong,
                        color = Rahal.colors.ink,
                    )
                },
            ) {
                RahalButton(
                    onClick = { pick = PickFor.Selected },
                    enabled = !state.busy,
                    modifier = Modifier.fillMaxWidth(),
                ) { Text(stringResource(R.string.ps_move_selected)) }
            }
        }
    }

    pick?.let { target ->
        SectionSheet(
            sections = state.sections,
            onDismiss = { pick = null },
            onPick = { id ->
                pick = null
                when (target) {
                    is PickFor.One -> state.choose(target.itemId, id)
                    PickFor.Selected -> state.moveSelected(id)
                }
            },
        )
    }

    if (state.confirmingRest) {
        ConfirmDialog(
            title = stringResource(R.string.ps_rest),
            body = stringResource(R.string.ps_rest_ask, list.total),
            confirm = stringResource(R.string.ps_rest_yes),
            onConfirm = { state.moveRest() },
            onDismiss = { state.dismissRest() },
            danger = false,
        )
    }
}

/** الجزء الأعلى: زر التوزيع التلقائي، سطر النتيجة، البحث، وأدوات التحديد. */
@Composable
private fun TopPart(state: PendingSectionsState) {
    Column(verticalArrangement = Arrangement.spacedBy(Rahal.space.sm)) {
        RahalButton(
            onClick = { state.autoSort() },
            enabled = !state.busy,
            modifier = Modifier.fillMaxWidth(),
        ) { Text(stringResource(R.string.ps_auto)) }
        Text(
            stringResource(R.string.ps_auto_hint),
            style = Rahal.type.caption,
            color = Rahal.colors.inkMuted,
        )
        when (val r = state.result) {
            is PendingSectionsState.Result.Sorted -> RahalCard(variant = CardVariant.Tinted, tone = StatusTone.Success) {
                Text(
                    stringResource(R.string.ps_sorted, r.r.byName, r.r.byGroup, r.r.waiting),
                    style = Rahal.type.bodyStrong,
                    color = Rahal.colors.success,
                )
            }
            is PendingSectionsState.Result.Moved -> RahalCard(variant = CardVariant.Tinted, tone = StatusTone.Success) {
                Text(
                    stringResource(R.string.ps_moved, r.n),
                    style = Rahal.type.bodyStrong,
                    color = Rahal.colors.success,
                )
            }
            null -> {}
        }
        LabeledField(
            label = stringResource(R.string.ps_search),
            value = state.query,
            onValueChange = { state.onQuery(it) },
        )
        if (state.list.items.isNotEmpty()) {
            Row(Modifier.fillMaxWidth(), verticalAlignment = Alignment.CenterVertically) {
                if (state.selecting) {
                    Checkbox(
                        checked = state.list.allSelected,
                        onCheckedChange = { state.toggleAll() },
                        colors = CheckboxDefaults.colors(checkedColor = Rahal.colors.brand),
                    )
                    Text(
                        stringResource(R.string.ps_select_all),
                        style = Rahal.type.body,
                        color = Rahal.colors.ink,
                        modifier = Modifier.weight(1f).clickable { state.toggleAll() },
                    )
                    RahalTextButton(onClick = { state.stopSelecting() }) {
                        Text(stringResource(R.string.ps_select_off))
                    }
                } else {
                    Spacer(Modifier.weight(1f))
                    RahalTextButton(onClick = { state.startSelecting() }) {
                        Text(stringResource(R.string.ps_select))
                    }
                }
            }
        }
    }
}

@Composable
private fun ItemRow(state: PendingSectionsState, item: PendingSyncItem, onPick: () -> Unit) {
    val list = state.list
    val sectionId = list.sectionFor(item)
    val sectionName = state.sections.firstOrNull { it.id == sectionId }?.name
        ?: item.suggestedSectionName?.takeIf { sectionId == item.suggestedSectionId }
        ?: ""
    val isSuggestion = sectionId.isNotEmpty() && sectionId == item.suggestedSectionId
    RahalCard(
        variant = CardVariant.Outlined,
        onClick = if (state.selecting) ({ state.toggle(item.id) }) else null,
    ) {
        Row(verticalAlignment = Alignment.CenterVertically) {
            if (state.selecting) {
                Checkbox(
                    checked = item.id in list.selected,
                    onCheckedChange = { state.toggle(item.id) },
                    colors = CheckboxDefaults.colors(checkedColor = Rahal.colors.brand),
                )
            }
            Column(Modifier.weight(1f)) {
                Text(
                    item.name,
                    style = Rahal.type.cardTitle,
                    color = Rahal.colors.ink,
                    maxLines = 2,
                    overflow = TextOverflow.Ellipsis,
                )
                val code = item.code?.takeIf { it.isNotBlank() }?.let { stringResource(R.string.ps_code, it) }
                Text(
                    listOfNotNull(code, priceLine(item.price, null)).joinToString(" · "),
                    style = Rahal.type.caption,
                    color = Rahal.colors.inkMuted,
                )
            }
        }
        if (!state.selecting) {
            Spacer(Modifier.height(Rahal.space.sm))
            Row(
                Modifier.fillMaxWidth(),
                verticalAlignment = Alignment.CenterVertically,
                horizontalArrangement = Arrangement.spacedBy(Rahal.space.sm),
            ) {
                RahalOutlineButton(onClick = onPick, enabled = !state.busy, modifier = Modifier.weight(1f)) {
                    Text(
                        if (sectionName.isEmpty()) {
                            stringResource(R.string.ps_pick)
                        } else if (isSuggestion) {
                            sectionName + " · " + stringResource(R.string.ps_suggested)
                        } else {
                            sectionName
                        },
                        maxLines = 1,
                        overflow = TextOverflow.Ellipsis,
                    )
                }
                RahalButton(
                    onClick = { state.moveOne(item.id) },
                    enabled = !state.busy && sectionId.isNotEmpty(),
                ) { Text(stringResource(R.string.ps_move)) }
            }
        }
    }
}

/** آخر القائمة: تحميل تلقائي للصفحة التالية، وزر «عرض المزيد» لإعادة المحاولة. */
@Composable
private fun Footer(state: PendingSectionsState) {
    if (!state.list.hasMore) return
    // يشتغل لما يوصل المستخدم هون، ويعيد نفسه بعد كل صفحة تنضاف.
    LaunchedEffect(state.list.items.size) { state.loadMore() }
    if (state.loadingMore) {
        LoadState(true, "")
    } else {
        RahalTextButton(onClick = { state.loadMore() }, modifier = Modifier.fillMaxWidth()) {
            Text(stringResource(R.string.ps_more))
        }
    }
}

/** ورقة اختيار القسم: قائمة أقسام السوق، والضغطة تختار وتغلق. */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
private fun SectionSheet(
    sections: List<PlatformSectionRef>,
    onDismiss: () -> Unit,
    onPick: (String) -> Unit,
) {
    ModalBottomSheet(
        onDismissRequest = onDismiss,
        sheetState = rememberModalBottomSheetState(skipPartiallyExpanded = true),
        containerColor = Rahal.colors.canvas,
        dragHandle = null,
    ) {
        SheetHeader(stringResource(R.string.ps_sheet_title), onClose = onDismiss)
        LazyColumn(Modifier.fillMaxWidth().padding(horizontal = Rahal.space.sm)) {
            items(sections, key = { it.id }) { s ->
                ListRow(title = s.name, onClick = { onPick(s.id) })
            }
        }
    }
}
