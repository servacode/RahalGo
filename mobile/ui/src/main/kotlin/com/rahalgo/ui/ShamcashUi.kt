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
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.material3.Icon
import androidx.compose.ui.draw.clip
import androidx.compose.ui.res.painterResource
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.layout.widthIn
import androidx.compose.foundation.layout.aspectRatio
import androidx.compose.foundation.clickable
import androidx.compose.foundation.border
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.ui.draw.shadow
import androidx.compose.ui.draw.drawBehind
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.lerp
import androidx.compose.ui.semantics.Role
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.sp
import androidx.compose.runtime.CompositionLocalProvider
import androidx.compose.ui.platform.LocalLayoutDirection
import androidx.compose.ui.unit.LayoutDirection
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
fun ShamcashAccountCard(info: ShamcashInfo, modifier: Modifier = Modifier, hero: Boolean = false) {
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

    val copyNumber = {
        clipboard.setText(AnnotatedString(info.accountNumber))
        Flash.ok(copiedText)
    }
    val saveQr = {
        val needPerm = Build.VERSION.SDK_INT < Build.VERSION_CODES.Q &&
            context.checkSelfPermission(Manifest.permission.WRITE_EXTERNAL_STORAGE) !=
            PackageManager.PERMISSION_GRANTED
        if (needPerm) permission.launch(Manifest.permission.WRITE_EXTERNAL_STORAGE) else doSave()
    }

    // **الباركودُ على «تذكرةٍ» بيضاءَ ثابتة** — يُمسح من شاشةٍ أخرى فلا تغمق أرضُه بالسمة الغامقة.
    val ticket: @Composable () -> Unit = {
        if (qr != null) {
            Box(Modifier.fillMaxWidth(), contentAlignment = Alignment.Center) {
                Box(
                    Modifier
                        .widthIn(max = 236.dp)
                        .fillMaxWidth()
                        .shadow(Rahal.elevation.raised, Rahal.shape.lg)
                        .clip(Rahal.shape.lg)
                        .background(Rahal.colors.onPanel)
                        .padding(Rahal.space.sm),
                ) {
                    AsyncImage(
                        model = qr,
                        contentDescription = stringResource(R.string.sham_qr_desc),
                        contentScale = ContentScale.Fit,
                        modifier = Modifier.fillMaxWidth().aspectRatio(1f),
                    )
                }
            }
        }
    }

    // **اسمُ صاحب الحساب ثمّ رقمُه في رقاقةٍ بأحرفٍ متباعدةٍ ومعها أيقونةُ نسخٍ** ثمّ الزرّان.
    val details: @Composable () -> Unit = {
        if (info.accountName.isNotBlank()) {
            Text(
                stringResource(R.string.sham_holder),
                color = Rahal.colors.inkMuted,
                style = Rahal.type.caption,
                modifier = Modifier.fillMaxWidth(),
                textAlign = TextAlign.Center,
            )
            Text(
                info.accountName,
                color = Rahal.colors.ink,
                style = Rahal.type.sectionTitle,
                fontWeight = FontWeight.Bold,
                modifier = Modifier.fillMaxWidth(),
                textAlign = TextAlign.Center,
            )
            Spacer(Modifier.height(Rahal.space.md))
        }
        Text(
            stringResource(R.string.sham_account_number),
            color = Rahal.colors.inkMuted,
            style = Rahal.type.caption,
            modifier = Modifier.fillMaxWidth(),
            textAlign = TextAlign.Center,
        )
        Spacer(Modifier.height(Rahal.space.xs))
        Row(
            Modifier
                .fillMaxWidth()
                .clip(Rahal.shape.md)
                .background(Rahal.colors.field)
                .border(1.dp, Rahal.colors.line, Rahal.shape.md)
                .padding(start = Rahal.space.md, top = Rahal.space.xs, bottom = Rahal.space.xs, end = Rahal.space.xs),
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.spacedBy(Rahal.space.sm),
        ) {
            // **الرقمُ أحرفٌ لاتينيّةٌ وأرقام** فيُعرض من اليسار بخطٍّ ثابتِ العرض، ويلتفّ على السطر التالي إن ضاق.
            CompositionLocalProvider(LocalLayoutDirection provides LayoutDirection.Ltr) {
                Text(
                    info.accountNumber,
                    color = Rahal.colors.ink,
                    style = Rahal.type.bodyStrong,
                    fontFamily = FontFamily.Monospace,
                    letterSpacing = 1.sp,
                    maxLines = 3,
                    modifier = Modifier.weight(1f),
                )
            }
            val copyDesc = stringResource(R.string.sham_copy)
            Box(
                Modifier
                    .size(44.dp)
                    .clip(CircleShape)
                    .background(Rahal.colors.brand)
                    .clickable(role = Role.Button, onClickLabel = copyDesc, onClick = copyNumber),
                contentAlignment = Alignment.Center,
            ) {
                Icon(
                    painterResource(R.drawable.ic_copy),
                    contentDescription = copyDesc,
                    tint = Rahal.colors.onBrand,
                    modifier = Modifier.size(Rahal.icon.xs),
                )
            }
        }
        Spacer(Modifier.height(Rahal.space.md))
        Row(horizontalArrangement = Arrangement.spacedBy(Rahal.space.md)) {
            RahalButton(onClick = copyNumber, modifier = Modifier.weight(1f), compact = qr != null) {
                Icon(painterResource(R.drawable.ic_copy), contentDescription = null, modifier = Modifier.size(Rahal.icon.xs))
                Spacer(Modifier.size(Rahal.space.xs))
                Text(stringResource(R.string.sham_copy), maxLines = 1)
            }
            if (qr != null) {
                RahalOutlineButton(onClick = saveQr, modifier = Modifier.weight(1f), compact = true) {
                    Icon(painterResource(R.drawable.ic_image), contentDescription = null, modifier = Modifier.size(Rahal.icon.xs))
                    Spacer(Modifier.size(Rahal.space.xs))
                    Text(stringResource(R.string.sham_save_qr), maxLines = 1)
                }
            }
        }
    }

    if (!hero) {
        RahalCard(modifier = modifier.fillMaxWidth(), variant = CardVariant.Outlined) {
            ticket()
            if (qr != null) Spacer(Modifier.height(Rahal.space.lg))
            details()
        }
        return
    }

    // **بطاقةُ الشحن**: رأسٌ بتدرّج العلامة (على نسق بطاقة الرصيد) فيه عنوانٌ وسطرٌ والتذكرة، وتحته التفاصيلُ.
    val top = Rahal.colors.brand
    val on = Rahal.colors.onBrand
    val bottom = if (Rahal.colors.dark) lerp(top, Rahal.colors.onPanel, 0.30f) else lerp(top, Rahal.colors.panel, 0.60f)
    Column(
        modifier
            .fillMaxWidth()
            .shadow(Rahal.elevation.raised, Rahal.shape.lg)
            .clip(Rahal.shape.lg)
            // **أرضُ التفاصيل فاتحة** (مراجعةُ المحاكي ٢٠٢٦-١١-٢٥) — `panel` لوحُ الخريطة الداكن، فكان الاسمُ
            // وزرُّ «حفظ صورة» لا يُقرآن عليه.
            .background(Rahal.colors.surface)
            .border(1.dp, Rahal.colors.line, Rahal.shape.lg),
    ) {
        Column(
            Modifier
                .fillMaxWidth()
                .background(Brush.linearGradient(listOf(top, bottom)))
                .drawBehind {
                    drawCircle(on.copy(alpha = 0.08f), radius = size.minDimension * 0.5f, center = Offset(size.width * 0.1f, size.height * 0.05f))
                    drawCircle(on.copy(alpha = 0.06f), radius = size.minDimension * 0.4f, center = Offset(size.width * 0.95f, size.height * 0.95f))
                }
                .padding(Rahal.space.lg),
        ) {
            Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(Rahal.space.md)) {
                IconBox(R.drawable.ic_wallet, on, size = Rahal.size.avatarSm)
                Column(Modifier.weight(1f)) {
                    Text(stringResource(R.string.sham_account_title), color = on, style = Rahal.type.sectionTitle, fontWeight = FontWeight.Bold)
                    Text(stringResource(R.string.sham_account_sub), color = on.copy(alpha = 0.88f), style = Rahal.type.caption)
                }
            }
            if (qr != null) Spacer(Modifier.height(Rahal.space.lg))
            ticket()
        }
        Column(Modifier.fillMaxWidth().padding(Rahal.space.lg)) { details() }
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

/** **كلمةُ حالة طلب الشحن** — ناطر · انشحنت · مرفوض (وسببُه سطرٌ تحتها). */
@Composable
private fun shamStatusText(r: ShamcashRequest): String = when (r.status) {
    "approved" -> stringResource(R.string.sham_st_approved)
    "rejected" -> stringResource(R.string.sham_st_rejected)
    else -> stringResource(R.string.sham_st_pending)
}

/** **قائمةُ طلبات الشحن بشام كاش** في المحفظة — الحالةُ رقاقةٌ بلونٍ وأيقونةٍ ونصّ، لا لوناً وحدَه. */
@Composable
fun ShamcashRequestsList(requests: List<ShamcashRequest>) {
    if (requests.isEmpty()) return
    SectionHeader(stringResource(R.string.sham_requests_title))
    RahalCard(padding = Rahal.space.none) {
        requests.forEachIndexed { i, r ->
            if (i > 0) RahalDivider(inset = Rahal.space.lg)
            val tone = when (r.status) {
                "approved" -> StatusTone.Success
                "rejected" -> StatusTone.Danger
                else -> StatusTone.Warning
            }
            val icon = when (r.status) {
                "approved" -> R.drawable.ic_check
                "rejected" -> R.drawable.ic_close
                else -> R.drawable.ic_time
            }
            Row(
                Modifier
                    .fillMaxWidth()
                    .padding(horizontal = Rahal.space.lg, vertical = Rahal.space.md),
                verticalAlignment = Alignment.CenterVertically,
                horizontalArrangement = Arrangement.spacedBy(Rahal.space.md),
            ) {
                Column(Modifier.weight(1f)) {
                    Text(money(r.amount), color = Rahal.colors.ink, style = Rahal.type.cardTitle)
                    CompositionLocalProvider(LocalLayoutDirection provides LayoutDirection.Ltr) {
                        Text(
                            r.txnRef,
                            color = Rahal.colors.inkMuted,
                            style = Rahal.type.caption,
                            fontFamily = FontFamily.Monospace,
                            maxLines = 1,
                            overflow = androidx.compose.ui.text.style.TextOverflow.Ellipsis,
                        )
                    }
                    val why = r.rejectReason
                    if (r.status == "rejected" && !why.isNullOrBlank()) {
                        Text(why, color = Rahal.colors.danger, style = Rahal.type.caption)
                    }
                    if (r.createdAt.isNotBlank()) {
                        Text(whenText(r.createdAt), color = Rahal.colors.inkMuted, style = Rahal.type.meta)
                    }
                }
                StatusChip(shamStatusText(r), tone, icon = icon)
            }
        }
    }
}

/** **رأسُ خطوةٍ مرقَّمة** — دائرةٌ بالرقم وعنوانُ الخطوة؛ ومحتواها تحتها. */
@Composable
private fun Step(number: String, title: String, last: Boolean = false, content: @Composable () -> Unit) {
    // **سلّمٌ عموديّ**: دائرةٌ مرقَّمة وخطٌّ رفيعٌ يصلها بالخطوة التالية، والمحتوى بجانبها.
    // **ولا `IntrinsicSize`** (مراجعةُ المحاكي ٢٠٢٦-١١-٢٥) — صورةُ الباركود تُحمَّل بعد القياس فكان يترك فراغاً
    // كبيراً تحت الخطوة؛ فالدوائرُ المرقّمةُ وحدها تكفي دليلاً.
    Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.spacedBy(Rahal.space.md)) {
        Column(Modifier.width(Rahal.size.iconBox), horizontalAlignment = Alignment.CenterHorizontally) {
            Box(
                Modifier
                    .size(Rahal.size.iconBox)
                    .clip(CircleShape)
                    .background(Rahal.colors.brand),
                contentAlignment = Alignment.Center,
            ) {
                Text(number, color = Rahal.colors.onBrand, style = Rahal.type.cardTitle)
            }
        }
        Column(Modifier.weight(1f).padding(bottom = if (last) Rahal.space.none else Rahal.space.xl)) {
            Box(Modifier.height(Rahal.size.iconBox), contentAlignment = Alignment.CenterStart) {
                Text(title, color = Rahal.colors.ink, style = Rahal.type.sectionTitle, fontWeight = FontWeight.Bold)
            }
            Spacer(Modifier.height(Rahal.space.sm))
            content()
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
        // **١ — حوّل** (الحسابُ والباركود)
        Step("١", stringResource(R.string.sham_step1)) {
            Text(stringResource(R.string.sham_step_hint), color = Rahal.colors.inkMuted, style = Rahal.type.body)
            Spacer(Modifier.height(Rahal.space.md))
            ShamcashAccountCard(info, hero = true)
        }

        // **٢ — اكتب رقم العملية** (المبلغ ورقمُ العملية والإيصال)
        Step("٢", stringResource(R.string.sham_step2)) {
            RahalCard(variant = CardVariant.Outlined) {
                LabeledField(
                    label = stringResource(R.string.sham_amount),
                    value = amount,
                    onValueChange = { amount = it.filter(Char::isDigit) },
                    keyboardType = KeyboardType.Number,
                    leadingIcon = R.drawable.ic_banknotes,
                    modifier = Modifier.fillMaxWidth(),
                )
                Text(
                    text = if (belowMin) stringResource(R.string.sham_amount_min, money(info.minTopup))
                    else stringResource(R.string.sham_amount_hint, money(info.minTopup)),
                    color = if (belowMin) Rahal.colors.danger else Rahal.colors.inkMuted,
                    style = Rahal.type.caption,
                )
                Spacer(Modifier.height(Rahal.space.lg))
                LabeledField(
                    label = stringResource(R.string.sham_txn),
                    value = txn,
                    onValueChange = { txn = it },
                    supporting = stringResource(R.string.sham_txn_hint),
                    modifier = Modifier.fillMaxWidth(),
                )
                Spacer(Modifier.height(Rahal.space.lg))
                RahalOutlineButton(
                    onClick = {
                        pick.launch(PickVisualMediaRequest(ActivityResultContracts.PickVisualMedia.ImageOnly))
                    },
                    enabled = !s.receiptBusy,
                    modifier = Modifier.fillMaxWidth(),
                ) {
                    Icon(painterResource(R.drawable.ic_photo), contentDescription = null, modifier = Modifier.size(Rahal.icon.xs))
                    Spacer(Modifier.size(Rahal.space.sm))
                    Text(stringResource(if (s.receiptId != null) R.string.sham_receipt_change else R.string.sham_receipt_add))
                }
                if (s.receiptId != null) {
                    Spacer(Modifier.height(Rahal.space.xs))
                    Text(stringResource(R.string.sham_receipt_attached), color = Rahal.colors.success, style = Rahal.type.caption)
                }
            }
        }

        // **٣ — أرسل**
        Step("٣", stringResource(R.string.sham_step3), last = true) {
            Text(stringResource(R.string.sham_step3_hint), color = Rahal.colors.inkMuted, style = Rahal.type.body)
            Spacer(Modifier.height(Rahal.space.md))
            RahalButton(
                onClick = { vm.submitShamcashTopup(value, txn.trim()) },
                enabled = !s.shamBusy && !s.receiptBusy && value >= maxOf(info.minTopup, 1) && txn.isNotBlank(),
                modifier = Modifier.fillMaxWidth(),
            ) { Text(stringResource(R.string.sham_send)) }
        }
        // **وفراغٌ لقرص الحديث العائم** (مراجعةُ المحاكي ٢٠٢٦-١١-٢٥) — كان يغطّي زرَّ «أرسلت».
        Spacer(Modifier.height(Rahal.space.xxl + LocalChatFabInset.current))
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
