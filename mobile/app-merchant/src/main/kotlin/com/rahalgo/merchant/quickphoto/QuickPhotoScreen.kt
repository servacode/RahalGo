package com.rahalgo.merchant.quickphoto

import androidx.activity.compose.BackHandler
import androidx.compose.foundation.Image
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ColumnScope
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.remember
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.asImageBitmap
import androidx.compose.ui.unit.dp
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import com.rahalgo.design.Rahal
import com.rahalgo.ui.cardSurface
import com.rahalgo.merchant.R
import com.rahalgo.shared.merchant.NoPhotoItem
import com.rahalgo.ui.LoadState
import com.rahalgo.ui.RahalButton
import com.rahalgo.ui.RahalTextButton
import com.rahalgo.ui.Tone
import com.rahalgo.ui.menu.priceLine
import com.rahalgo.ui.rememberCameraShot

/**
 * Fast photo mode screen (owner decision 2026-11-06): the item name large, the camera one tap away,
 * a preview with "save and continue" / "retake", then the next item appears by itself.
 */
@Composable
fun QuickPhotoScreen(state: QuickPhotoState) {
    BackHandler { state.close() }
    val shoot = rememberCameraShot(onDenied = {}) { state.capture(it) }
    val q = state.queue

    Column(Modifier.fillMaxSize().padding(Rahal.space.lg)) {
        Row(Modifier.fillMaxWidth(), verticalAlignment = Alignment.CenterVertically) {
            Text(
                stringResource(R.string.qp_title),
                style = MaterialTheme.typography.titleMedium,
                fontWeight = FontWeight.Bold,
                modifier = Modifier.weight(1f),
            )
            if (q != null && q.total > 0) {
                Text(
                    stringResource(R.string.qp_counter, q.position, q.total),
                    style = MaterialTheme.typography.titleSmall,
                    color = Rahal.colors.brand,
                )
            }
            RahalTextButton(onClick = { state.close() }) { Text(stringResource(R.string.qp_close)) }
        }
        // **شريطُ التقدّم** (مراجعةُ التصميم ٢٠٢٦-١١-٠٩) — يُرى كم بقي قبل أن يُقرأ الرقم.
        if (q != null && q.total > 0) {
            Spacer(Modifier.height(Rahal.space.sm))
            androidx.compose.material3.LinearProgressIndicator(
                progress = { (q.position.toFloat() / q.total).coerceIn(0f, 1f) },
                modifier = Modifier.fillMaxWidth().height(6.dp).clip(Rahal.shape.pill),
                color = Rahal.colors.brand,
                trackColor = Rahal.colors.line,
                drawStopIndicator = {},
            )
        }
        UploadBanner(state)
        Spacer(Modifier.height(Rahal.space.md))

        when {
            state.loading || q == null -> LoadState(true, "")
            q.finished -> Done(state, q.total)
            else -> Capture(state, q.current!!, shoot)
        }
    }
}

@Composable
private fun UploadBanner(state: QuickPhotoState) {
    if (state.uploading > 0) {
        Text(
            stringResource(R.string.qp_uploading, state.uploading),
            style = MaterialTheme.typography.bodySmall,
            color = Rahal.colors.ink,
        )
    }
    if (state.failed.isNotEmpty()) {
        Row(Modifier.fillMaxWidth(), verticalAlignment = Alignment.CenterVertically) {
            Text(
                stringResource(R.string.qp_failed, state.failed.size),
                style = MaterialTheme.typography.bodySmall,
                color = Rahal.colors.danger,
                modifier = Modifier.weight(1f),
            )
            RahalTextButton(onClick = { state.retryFailed() }, tone = Tone.Danger) {
                Text(stringResource(R.string.qp_retry))
            }
        }
    }
}

@Composable
private fun ColumnScope.Capture(state: QuickPhotoState, item: NoPhotoItem, shoot: () -> Unit) {
    Text(
        item.name,
        style = MaterialTheme.typography.headlineMedium,
        fontWeight = FontWeight.Bold,
        textAlign = TextAlign.Center,
        modifier = Modifier.fillMaxWidth(),
    )
    Text(
        listOf(item.sectionName, priceLine(item.price, item.priceUsd)).filter { it.isNotBlank() }.joinToString(" · "),
        style = MaterialTheme.typography.bodySmall,
        color = Rahal.colors.inkMuted,
        textAlign = TextAlign.Center,
        modifier = Modifier.fillMaxWidth(),
    )
    Spacer(Modifier.height(Rahal.space.md))
    val shot = state.captured
    Box(
        Modifier
            .weight(1f)
            .fillMaxWidth()
            .cardSurface(Rahal.shape.lg)
            .padding(Rahal.space.sm),
        contentAlignment = Alignment.Center,
    ) {
        if (shot != null) {
            val bmp = remember(shot) { android.graphics.BitmapFactory.decodeByteArray(shot, 0, shot.size) }
            if (bmp != null) {
                Image(
                    bmp.asImageBitmap(),
                    contentDescription = item.name,
                    contentScale = ContentScale.Fit,
                    modifier = Modifier.fillMaxSize(),
                )
            }
        } else {
            Column(horizontalAlignment = Alignment.CenterHorizontally) {
                com.rahalgo.ui.IconTile(
                    icon = com.rahalgo.ui.R.drawable.ic_photo,
                    tint = Rahal.colors.brand,
                    size = 72.dp,
                )
                Spacer(Modifier.height(Rahal.space.md))
                Text(
                    stringResource(R.string.qp_hint),
                    style = MaterialTheme.typography.bodyMedium,
                    color = Rahal.colors.inkMuted,
                    textAlign = TextAlign.Center,
                )
            }
        }
    }
    Spacer(Modifier.height(Rahal.space.md))
    if (shot == null) {
        RahalButton(onClick = shoot, modifier = Modifier.fillMaxWidth()) { Text(stringResource(R.string.qp_shoot)) }
    } else {
        Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.spacedBy(Rahal.space.sm)) {
            RahalButton(onClick = { state.retake(); shoot() }, tone = Tone.Accent, modifier = Modifier.weight(1f)) {
                Text(stringResource(R.string.qp_retake))
            }
            RahalButton(onClick = { state.saveAndNext() }, tone = Tone.Success, modifier = Modifier.weight(2f)) {
                Text(stringResource(R.string.qp_save_next))
            }
        }
    }
    RahalTextButton(onClick = { state.skip() }, modifier = Modifier.fillMaxWidth()) {
        Text(stringResource(R.string.qp_skip))
    }
}

@Composable
private fun Done(state: QuickPhotoState, total: Int) {
    Column(Modifier.fillMaxSize(), verticalArrangement = Arrangement.Center, horizontalAlignment = Alignment.CenterHorizontally) {
        com.rahalgo.ui.IconTile(
            icon = com.rahalgo.ui.R.drawable.ic_check,
            tint = Rahal.colors.success,
            size = 72.dp,
        )
        Spacer(Modifier.height(Rahal.space.lg))
        Text(
            stringResource(if (total == 0) R.string.qp_none else R.string.qp_done),
            style = MaterialTheme.typography.titleLarge,
            fontWeight = FontWeight.Bold,
            textAlign = TextAlign.Center,
        )
        Spacer(Modifier.height(Rahal.space.lg))
        RahalButton(onClick = { state.close() }) { Text(stringResource(R.string.qp_close)) }
    }
}
