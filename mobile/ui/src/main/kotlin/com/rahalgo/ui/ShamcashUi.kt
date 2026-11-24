package com.rahalgo.ui

import android.Manifest
import android.content.ContentValues
import android.content.Context
import android.content.pm.PackageManager
import android.os.Build
import android.os.Environment
import android.provider.MediaStore
import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.result.PickVisualMediaRequest
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.platform.LocalClipboardManager
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.AnnotatedString
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.unit.dp
import coil3.compose.AsyncImage
import com.rahalgo.design.Rahal
import com.rahalgo.shared.customer.ShamcashInfo
import com.rahalgo.shared.customer.ShamcashRequest
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext

/**
 * **بطاقةُ حساب شام كاش** — الباركودُ واسمُ صاحب الحساب ورقمُه وزرّان كبيران: نسخُ الرقم بلمسة،
 * وحفظُ صورة الباركود بالمعرض. **تُستعمل في شحن المحفظة وفي الدفع عند الباب** فلا تفترق نسختان.
 */
@Composable
fun ShamcashAccountCard(info: ShamcashInfo, modifier: Modifier = Modifier) {
    val context = LocalContext.current
    val clipboard = LocalClipboardManager.current
    val scope = rememberCoroutineScope()
    val qr = AppCore.get().media(info.qrUrl)
    val copiedText = stringResource(R.string.sham_copied)
    val savedText = stringResource(R.string.sham_qr_saved)
    val failedText = stringResource(R.string.sham_qr_failed)
    val permText = stringResource(R.string.sham_qr_perm)

    fun doSave() {
        val url = qr ?: return
        scope.launch {
            val ok = withContext(Dispatchers.IO) {
                val bytes = AppCore.get().api.bytes(url)
                bytes.isNotEmpty() && saveQrToGallery(context, bytes)
            }
            if (ok) Flash.ok(savedText) else Flash.fail(failedText)
        }
    }

    // **على أندرويد ٩ وما دون** الحفظُ بالمعرض يحتاج إذنَ الكتابة — يُطلب عند الضغطة.
    val permission = rememberLauncherForActivityResult(ActivityResultContracts.RequestPermission()) { granted ->
        if (granted) doSave() else Flash.fail(permText)
    }

    RahalCard(modifier = modifier.fillMaxWidth(), variant = CardVariant.Outlined) {
        if (qr != null) {
            Column(Modifier.fillMaxWidth(), horizontalAlignment = Alignment.CenterHorizontally) {
                AsyncImage(
                    model = qr,
                    contentDescription = null,
                    contentScale = ContentScale.Fit,
                    modifier = Modifier.size(240.dp),
                )
            }
            Spacer(Modifier.height(Rahal.space.md))
        }
        if (info.accountName.isNotBlank()) {
            Text(stringResource(R.string.sham_holder), color = Rahal.colors.inkMuted, style = Rahal.type.caption)
            Text(
                info.accountName,
                color = Rahal.colors.ink,
                style = Rahal.type.bodyLarge,
                fontWeight = FontWeight.Bold,
            )
            Spacer(Modifier.height(Rahal.space.sm))
        }
        Text(stringResource(R.string.sham_account_number), color = Rahal.colors.inkMuted, style = Rahal.type.caption)
        Text(
            info.accountNumber,
            color = Rahal.colors.ink,
            style = Rahal.type.sectionTitle,
            fontWeight = FontWeight.Bold,
        )
        Spacer(Modifier.height(Rahal.space.md))
        RahalButton(
            onClick = {
                clipboard.setText(AnnotatedString(info.accountNumber))
                Flash.ok(copiedText)
            },
            modifier = Modifier.fillMaxWidth(),
        ) { Text(stringResource(R.string.sham_copy)) }
        if (qr != null) {
            Spacer(Modifier.height(Rahal.space.sm))
            RahalOutlineButton(
                onClick = {
                    val needPerm = Build.VERSION.SDK_INT < Build.VERSION_CODES.Q &&
                        context.checkSelfPermission(Manifest.permission.WRITE_EXTERNAL_STORAGE) !=
                        PackageManager.PERMISSION_GRANTED
                    if (needPerm) permission.launch(Manifest.permission.WRITE_EXTERNAL_STORAGE) else doSave()
                },
                modifier = Modifier.fillMaxWidth(),
            ) { Text(stringResource(R.string.sham_save_qr)) }
        }
    }
}

/**
 * **يحفظ صورةَ الباركود بمعرض الهاتف** (`Pictures/RahalGo`) — بلا إذنٍ على أندرويد ١٠ وما فوق
 * (`MediaStore` + `IS_PENDING`)، وبإذن الكتابة تحتها. **والفاشلُ ⇒ `false`.**
 */
fun saveQrToGallery(context: Context, bytes: ByteArray): Boolean = runCatching {
    val png = bytes.size > 4 && bytes[0] == 0x89.toByte() && bytes[1] == 'P'.code.toByte()
    val mime = if (png) "image/png" else "image/jpeg"
    val name = "rahalgo_shamcash_" + System.currentTimeMillis() + if (png) ".png" else ".jpg"
    val resolver = context.contentResolver
    if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.Q) {
        val values = ContentValues().apply {
            put(MediaStore.Images.Media.DISPLAY_NAME, name)
            put(MediaStore.Images.Media.MIME_TYPE, mime)
            put(MediaStore.Images.Media.RELATIVE_PATH, Environment.DIRECTORY_PICTURES + "/RahalGo")
            put(MediaStore.Images.Media.IS_PENDING, 1)
        }
        val uri = resolver.insert(MediaStore.Images.Media.EXTERNAL_CONTENT_URI, values) ?: return@runCatching false
        try {
            resolver.openOutputStream(uri)?.use { it.write(bytes) } ?: error("no stream")
            val done = ContentValues().apply { put(MediaStore.Images.Media.IS_PENDING, 0) }
            resolver.update(uri, done, null, null)
        } catch (e: Exception) {
            resolver.delete(uri, null, null)
            throw e
        }
        true
    } else {
        @Suppress("DEPRECATION")
        val dir = java.io.File(Environment.getExternalStoragePublicDirectory(Environment.DIRECTORY_PICTURES), "RahalGo")
        if (!dir.exists() && !dir.mkdirs()) return@runCatching false
        val f = java.io.File(dir, name)
        f.writeBytes(bytes)
        android.media.MediaScannerConnection.scanFile(context, arrayOf(f.absolutePath), arrayOf(mime), null)
        true
    }
}.getOrDefault(false)

/** **كلمةُ حالة طلب الشحن** — ناطر · انشحنت · مرفوض مع سببه. */
@Composable
private fun shamStatusText(r: ShamcashRequest): String {
    val why = r.rejectReason
    return when (r.status) {
        "approved" -> stringResource(R.string.sham_st_approved)
        "rejected" ->
            if (!why.isNullOrBlank()) stringResource(R.string.sham_st_rejected_why, why)
            else stringResource(R.string.sham_st_rejected)
        else -> stringResource(R.string.sham_st_pending)
    }
}

/** **قائمةُ طلبات الشحن بشام كاش** في المحفظة. */
@Composable
fun ShamcashRequestsList(requests: List<ShamcashRequest>) {
    if (requests.isEmpty()) return
    SectionHeader(stringResource(R.string.sham_requests_title))
    RahalCard(padding = Rahal.space.none) {
        requests.forEachIndexed { i, r ->
            if (i > 0) RahalDivider(inset = Rahal.space.lg)
            val color = when (r.status) {
                "approved" -> Rahal.colors.success
                "rejected" -> Rahal.colors.danger
                else -> Rahal.colors.inkMuted
            }
            Column(
                Modifier
                    .fillMaxWidth()
                    .padding(horizontal = Rahal.space.lg, vertical = Rahal.space.md),
            ) {
                Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween) {
                    Text(money(r.amount), style = Rahal.type.cardTitle)
                    Text(shamStatusText(r), color = color, style = Rahal.type.label)
                }
                Text(r.txnRef, color = Rahal.colors.inkMuted, style = Rahal.type.caption)
            }
        }
    }
}

/**
 * **شاشةُ الشحن بشام كاش** — بطاقةُ الحساب ثمّ نموذجُ: المبلغ · رقمُ العملية · صورةُ الإيصال (اختياريّة).
 */
@Composable
fun ShamcashTopupScreen(vm: WalletViewModel, info: ShamcashInfo, onBack: () -> Unit) {
    androidx.activity.compose.BackHandler(onBack = onBack)
    val context = LocalContext.current
    val s = vm.state
    var amount by rememberSaveable { mutableStateOf("") }
    var txn by rememberSaveable { mutableStateOf("") }
    val value = amount.trim().toLongOrNull() ?: 0
    val belowMin = value in 1 until info.minTopup
    val pick = rememberLauncherForActivityResult(ActivityResultContracts.PickVisualMedia()) { uri ->
        if (uri != null) {
            runCatching { readScaledImage(context, uri) }.getOrNull()?.let(vm::uploadShamcashReceipt)
        }
    }
    Column(
        Modifier
            .fillMaxSize()
            .verticalScroll(rememberScrollState())
            .padding(Rahal.space.lg),
    ) {
        PageHeader(stringResource(R.string.sham_topup_title), onBack)
        Spacer(Modifier.height(Rahal.space.md))
        if (s.shamError.isNotEmpty()) {
            Note(s.shamError, Rahal.colors.danger)
            Spacer(Modifier.height(Rahal.space.sm))
        }
        if (s.shamDone) {
            Note(stringResource(R.string.sham_sent), Rahal.colors.success)
            Spacer(Modifier.height(Rahal.space.md))
            RahalButton(onClick = onBack, modifier = Modifier.fillMaxWidth()) {
                Text(stringResource(R.string.sham_back))
            }
            return@Column
        }
        Text(stringResource(R.string.sham_step_hint), color = Rahal.colors.inkMuted, style = Rahal.type.body)
        Spacer(Modifier.height(Rahal.space.md))
        ShamcashAccountCard(info)
        Spacer(Modifier.height(Rahal.space.lg))
        LabeledField(
            label = stringResource(R.string.sham_amount),
            value = amount,
            onValueChange = { amount = it.filter(Char::isDigit) },
            keyboardType = KeyboardType.Number,
            leadingIcon = R.drawable.ic_banknotes,
        )
        Text(
            text = if (belowMin) stringResource(R.string.sham_amount_min, money(info.minTopup))
            else stringResource(R.string.sham_amount_hint, money(info.minTopup)),
            color = if (belowMin) Rahal.colors.danger else Rahal.colors.inkMuted,
            style = Rahal.type.caption,
        )
        Spacer(Modifier.height(Rahal.space.sm))
        LabeledField(
            label = stringResource(R.string.sham_txn),
            value = txn,
            onValueChange = { txn = it },
            supporting = stringResource(R.string.sham_txn_hint),
        )
        Spacer(Modifier.height(Rahal.space.sm))
        RahalOutlineButton(
            onClick = {
                pick.launch(PickVisualMediaRequest(ActivityResultContracts.PickVisualMedia.ImageOnly))
            },
            enabled = !s.receiptBusy,
            modifier = Modifier.fillMaxWidth(),
        ) {
            Text(stringResource(if (s.receiptId != null) R.string.sham_receipt_change else R.string.sham_receipt_add))
        }
        if (s.receiptId != null) {
            Text(stringResource(R.string.sham_receipt_attached), color = Rahal.colors.success, style = Rahal.type.caption)
        }
        Spacer(Modifier.height(Rahal.space.md))
        RahalButton(
            onClick = { vm.submitShamcashTopup(value, txn.trim()) },
            enabled = !s.shamBusy && !s.receiptBusy && value >= maxOf(info.minTopup, 1) && txn.isNotBlank(),
            modifier = Modifier.fillMaxWidth(),
        ) { Text(stringResource(R.string.sham_send)) }
        Spacer(Modifier.height(Rahal.space.xxl))
    }
}

/** **معلوماتُ شام كاش مخبّأةٌ لحظاتٍ** — بطاقاتُ الطلبات الكثيرةُ لا تنادي الخادمَ كلٌّ لنفسها. */
private object ShamcashInfoCache {
    var info by mutableStateOf<ShamcashInfo?>(null)
    var at = 0L
}

/** **معلوماتُ حساب شام كاش** — و`null` حتى تصل (أو إن تعذّر الجلب). */
@Composable
fun rememberShamcashInfo(): ShamcashInfo? {
    LaunchedEffect(Unit) {
        val now = System.currentTimeMillis()
        if (ShamcashInfoCache.info == null || now - ShamcashInfoCache.at > 30_000) {
            runCatching { com.rahalgo.shared.customer.CustomerApi(AppCore.get().api).shamcash() }
                .getOrNull()
                ?.let {
                    ShamcashInfoCache.info = it.info
                    ShamcashInfoCache.at = now
                }
        }
    }
    return ShamcashInfoCache.info
}

/** **«ادفع للكابتن بشام كاش»** — يظهر والكابتنُ عند الباب والزبونُ اختار شام كاش. */
@Composable
fun ShamcashDoorCard(amount: Long) {
    val info = rememberShamcashInfo() ?: return
    RahalCard(modifier = Modifier.padding(top = Rahal.space.md), variant = CardVariant.Outlined) {
        Text(
            stringResource(R.string.sham_pay_title),
            color = Rahal.colors.ink,
            style = Rahal.type.label,
            fontWeight = FontWeight.Bold,
        )
        Text(
            stringResource(R.string.sham_pay_amount, money(amount)),
            color = Rahal.colors.brand,
            style = Rahal.type.bodyLarge,
            fontWeight = FontWeight.Bold,
        )
        Spacer(Modifier.height(Rahal.space.sm))
        ShamcashAccountCard(info)
    }
}
