package com.rahalgo.ui

import android.content.Context
import android.content.Intent
import android.net.Uri
import androidx.compose.foundation.Canvas
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.semantics.Role
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.rahalgo.design.Rahal
import com.rahalgo.shared.model.Platform

/**
 * ══════════════════════════════════════════════════════════════════════
 * **تواصل معنا — ولا رقمَ مكتوبٌ في الشيفرة**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (قاعدةُ المالك: «لا أريد أن تكتب اسم المنصّة بأيّ مكانٍ أبدا» — وكذلك
 *  رقمُها وعنوانُها.)
 *
 * **وما كُتب في شيفرةٍ لا يُبدَّل إلّا بنشر** — فيبقى الرقمُ القديمُ
 * معروضاً شهراً، **ومن اتّصل به لم يجد أحدا.**
 *
 * # وفارغُه يُحذف لا يُعرض
 *
 * **وسطرٌ يقول «الهاتف: —» أسوأُ من غيابه**: يُقرأ عطباً في المنصّة لا
 * حقلاً لم يُملأ بعد. **وأيقونةٌ لا تفتح شيئاً عطبٌ ظاهر.**
 *
 * # ويُضغط فيقع الفعل
 *
 * **رقمٌ يُقرأ ثمّ يُكتب بالإصبع في لوحة الاتّصال عملٌ زائد** — والسائقُ
 * يفتح هذه الشاشةَ وهو في الشارع.
 *
 * # وأيقوناتٌ لا أرقامٌ ولا روابط (قرارُ المالك ٢٠٢٦-١٠-٠٥)
 *
 * **رابطُ فيسبوك مكتوباً سطرٌ لاتينيٌّ طويلٌ لا يقرؤه أحد** — والأيقونةُ
 * تقول ما هي وتفتحه. **والرقمُ لا يُطبع**: زرّا اتّصالٍ وواتساب يفعلان
 * ما كان سيفعله بالرقم بعد أن ينسخه.
 */
@Composable
fun ContactPage(vm: PagesViewModel) {
    val p = vm.platform
    if (p == null) {
        LoadState(vm.busy, vm.error) { vm.load(force = true) }
        return
    }

    Screen {
        ScreenTitle(
            stringResource(R.string.menu_contact),
            stringResource(R.string.contact_hint),
        )
        if (!hasContact(p)) {
            Empty(stringResource(R.string.contact_none))
            return@Screen
        }
        ContactSection(p)
    }
}

/**
 * ══════════════════════════════════════════════════════════════════════
 * **قسمُ التواصل — واحدٌ لصفحتين** (قرارُ المالك ٢٠٢٦-١٠-٠٥)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **«تواصل معنا» و«أين تجدنا» في «من نحن» يعرضان الشيءَ نفسَه** —
 * ونسختان منه تفترقان يومَ تُصلَح إحداهما. **فقطعةٌ واحدةٌ تُنادى مرّتين.**
 *
 * أيقوناتُ التواصل ← بطاقةُ العنوان ← معاينةُ الخريطة. **وكلُّ جزءٍ
 * فارغٍ يسقط وحدَه.**
 */
@Composable
fun ContactSection(p: Platform) {
    val context = LocalContext.current
    val wa = waMeDigits(p.social.whatsapp.ifBlank { p.supportPhone })
    val ways = buildList {
        if (p.supportPhone.isNotBlank()) {
            add(Way(R.drawable.ic_phone, R.string.contact_call, "tel:" + p.supportPhone.trim()))
        }
        if (wa.isNotEmpty()) add(Way(R.drawable.ic_whatsapp, R.string.contact_whatsapp, "https://wa.me/" + wa))
        socialUrl(p.social.facebook)?.let { add(Way(R.drawable.ic_facebook, R.string.contact_facebook, it)) }
        socialUrl(p.social.instagram)?.let { add(Way(R.drawable.ic_instagram, R.string.contact_instagram, it)) }
        socialUrl(p.social.telegram, telegram = true)?.let {
            add(Way(R.drawable.ic_telegram, R.string.contact_telegram, it))
        }
    }

    if (ways.isNotEmpty()) {
        // **صفٌّ واحدٌ في الوسط** — والاتّصالُ أوّلاً في جهة البداية،
        // **وهو ما يُطلب أكثر.**
        Row(
            Modifier.fillMaxWidth().padding(vertical = 6.dp),
            horizontalArrangement = Arrangement.spacedBy(14.dp, Alignment.CenterHorizontally),
        ) {
            ways.forEach { w -> RoundIcon(w) { openUrl(context, w.url) } }
        }
    }

    val address = formatAddress(p.address)
    if (address.isNotEmpty()) {
        Spacer(Modifier.height(12.dp))
        AddressCard(address)
    }

    val point = parseLatLng(p.location)
    if (point != null) {
        Spacer(Modifier.height(12.dp))
        MapPreview { openMap(context, point, p.name) }
    }
}

/** **أفيه ما يُعرض؟** — وإلّا قيل «لم تُضبط» بدل صفحةٍ بيضاء. */
fun hasContact(p: Platform): Boolean =
    p.supportPhone.isNotBlank() || p.address.isNotBlank() || parseLatLng(p.location) != null ||
        listOf(p.social.whatsapp, p.social.facebook, p.social.instagram, p.social.telegram)
            .any { it.isNotBlank() }

/** **بابُ تواصلٍ واحد** — أيقونتُه ووصفُها لقارئ الشاشة ورابطُه. */
private data class Way(val icon: Int, val label: Int, val url: String)

/**
 * **أيقونةٌ مستديرة** — أرضٌ باهتةٌ من لون العلامة والرسمُ بلونها.
 *
 * **وألوانُ الشعارات التجاريّة لا تُكتب هنا** — لونٌ ثابتٌ خارج التوكنز
 * يُكسر في السمة الغامقة. **والوصفُ لقارئ الشاشة** إذ لا نصَّ ظاهراً.
 */
@Composable
private fun RoundIcon(w: Way, onClick: () -> Unit) {
    val label = stringResource(w.label)
    Box(
        Modifier
            .size(52.dp)
            .clip(CircleShape)
            .background(Rahal.colors.brand.copy(alpha = 0.12f))
            .clickable(onClickLabel = label, role = Role.Button, onClick = onClick),
        contentAlignment = Alignment.Center,
    ) {
        Icon(
            painter = painterResource(w.icon),
            contentDescription = label,
            tint = Rahal.colors.brand,
            modifier = Modifier.size(24.dp),
        )
    }
}

/** **بطاقةُ العنوان** — دبّوسٌ في دائرةٍ، واسمُ الحقل باهتاً فوق العنوان. */
@Composable
private fun AddressCard(address: String) {
    Card {
        Row(verticalAlignment = Alignment.CenterVertically) {
            Box(
                Modifier
                    .size(40.dp)
                    .clip(CircleShape)
                    .background(Rahal.colors.brand.copy(alpha = 0.12f)),
                contentAlignment = Alignment.Center,
            ) {
                Icon(
                    painter = painterResource(R.drawable.ic_pin),
                    contentDescription = null,
                    tint = Rahal.colors.brand,
                    modifier = Modifier.size(20.dp),
                )
            }
            Spacer(Modifier.width(12.dp))
            Column(Modifier.weight(1f)) {
                Text(
                    text = stringResource(R.string.contact_address),
                    color = Rahal.colors.inkMuted,
                    style = MaterialTheme.typography.bodySmall,
                )
                Spacer(Modifier.height(2.dp))
                Text(
                    text = address,
                    color = Rahal.colors.ink,
                    fontWeight = FontWeight.Medium,
                    style = MaterialTheme.typography.bodyLarge,
                )
            }
        }
    }
}

/**
 * ══════════════════════════════════════════════════════════════════════
 * **معاينةُ الخريطة — بطاقةٌ تُضغط لا خريطةٌ حيّة**
 * ══════════════════════════════════════════════════════════════════════
 *
 * **وحدةُ الخرائط (`:map`) تعتمد على هذه الوحدة** — فلا تُستورد منها
 * بلا حلقة. **وخريطةٌ حيّةٌ في صفحةِ نصٍّ تُحمِّل بلاطاً على حزمةِ من في
 * الشارع** ليرى نقطةً واحدة.
 *
 * **فشبكةُ شوارعَ مرسومةٌ ودبّوسٌ في وسطها** — تُقرأ «خريطة» من النظرة،
 * **والضغطةُ تفتح تطبيقَ الخرائط على الموضع** حيث التنقّلُ الحقيقيّ.
 */
@Composable
private fun MapPreview(onClick: () -> Unit) {
    val open = stringResource(R.string.contact_map_open)
    val grid = Rahal.colors.line
    val road = Rahal.colors.brand.copy(alpha = 0.18f)
    Column(
        Modifier
            .fillMaxWidth()
            .clip(Rahal.shape.md)
            .background(Rahal.colors.brand.copy(alpha = 0.06f))
            .clickable(onClickLabel = open, role = Role.Button, onClick = onClick),
    ) {
        Box(Modifier.fillMaxWidth().height(140.dp), contentAlignment = Alignment.Center) {
            Canvas(Modifier.fillMaxSize()) {
                val step = 28.dp.toPx()
                var x = step / 2
                while (x < size.width) {
                    drawLine(grid, Offset(x, 0f), Offset(x, size.height), strokeWidth = 1f)
                    x += step
                }
                var y = step / 2
                while (y < size.height) {
                    drawLine(grid, Offset(0f, y), Offset(size.width, y), strokeWidth = 1f)
                    y += step
                }
                // **شارعان عريضان يتقاطعان قرب الدبّوس** — فتُقرأ مدينةً لا ورقةَ مربّعات.
                drawLine(road, Offset(0f, size.height * 0.62f), Offset(size.width, size.height * 0.38f), 10.dp.toPx())
                drawLine(road, Offset(size.width * 0.42f, 0f), Offset(size.width * 0.58f, size.height), 8.dp.toPx())
            }
            Box(
                Modifier
                    .size(44.dp)
                    .clip(CircleShape)
                    .background(Rahal.colors.accent),
                contentAlignment = Alignment.Center,
            ) {
                Icon(
                    painter = painterResource(R.drawable.ic_pin),
                    contentDescription = null,
                    tint = Color.White,
                    modifier = Modifier.size(24.dp),
                )
            }
        }
        Row(
            Modifier.fillMaxWidth().padding(horizontal = 14.dp, vertical = 12.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Icon(
                painter = painterResource(R.drawable.ic_map),
                contentDescription = null,
                tint = Rahal.colors.brand,
                modifier = Modifier.size(18.dp),
            )
            Spacer(Modifier.width(8.dp))
            Text(
                text = stringResource(R.string.contact_map),
                color = Rahal.colors.ink,
                style = MaterialTheme.typography.bodyMedium,
                modifier = Modifier.weight(1f),
            )
            Text(
                text = open,
                color = Rahal.colors.brand,
                fontWeight = FontWeight.Bold,
                style = MaterialTheme.typography.bodyMedium,
            )
        }
    }
}

/** **يفتح الرابط** — وفشلُه صامت: لا تطبيقَ يفتحه، فلا شيء يُعرض. */
private fun openUrl(context: Context, url: String) {
    runCatching { context.startActivity(Intent(Intent.ACTION_VIEW, Uri.parse(url))) }
}

/**
 * **يفتح الخرائط على الموضع** — `geo:` أوّلاً، **ومن لا تطبيقَ خرائطَ عنده
 * يُفتح له رابطُ خرائط غوغل في المتصفّح** بدل ضغطةٍ لا تفعل شيئا.
 */
private fun openMap(context: Context, point: Pair<Double, Double>, label: String) {
    runCatching {
        context.startActivity(Intent(Intent.ACTION_VIEW, Uri.parse(geoUri(point, label))))
    }.onFailure { openUrl(context, mapsWebUrl(point)) }
}

// ══════════════════════════════════════════════════════════════════════
// **دوالُّ صافيةٌ — تُقاس بلا جهاز** (`ContactLinksTest`)
// ══════════════════════════════════════════════════════════════════════

/**
 * **رقمُ wa.me** — أرقامٌ وحدَها بمفتاح الدولة، والفارغُ لا رابطَ له.
 *
 * **و`wa.me` لا يقبل الصفرَ المحلّيّ**: `0912345678` يفتح محادثةً مع
 * رقمٍ لا وجودَ له. **فيصير `963912345678`**، و`00963…` و`+963…` كذلك.
 */
fun waMeDigits(raw: String): String {
    val d = raw.filter(Char::isDigit)
    return when {
        d.isEmpty() -> ""
        d.startsWith("00") -> d.drop(2)
        d.length == 10 && d.startsWith("09") -> "963" + d.drop(1)
        d.length == 9 && d.startsWith("9") -> "963" + d
        else -> d
    }
}

/**
 * **رابطُ حسابٍ اجتماعيّ صالحٌ للفتح** — و`null` للفارغ.
 *
 * **ومن كتب في اللوحة `facebook.com/x` بلا بروتوكول** يُكمَل له، **و`@اسم`
 * في تلغرام يصير `t.me/اسم`** — فلا تُفتح الأيقونةُ على لا شيء.
 */
fun socialUrl(raw: String, telegram: Boolean = false): String? {
    val v = raw.trim()
    if (v.isEmpty()) return null
    if (v.contains("://")) return v
    if (telegram && v.startsWith("@")) return "https://t.me/" + v.drop(1)
    return "https://" + v
}

/** **`"lat,lng"` إلى نقطة** — و`null` لما لا يُحلَّل أو يقع خارج الكرة. */
fun parseLatLng(raw: String): Pair<Double, Double>? {
    val parts = raw.split(',').map { it.trim() }
    if (parts.size != 2) return null
    val lat = parts[0].toDoubleOrNull() ?: return null
    val lng = parts[1].toDoubleOrNull() ?: return null
    if (lat !in -90.0..90.0 || lng !in -180.0..180.0) return null
    return lat to lng
}

/**
 * **رابطُ `geo:` بدبّوسٍ واسم** — `geo:lat,lng?q=lat,lng(الاسم)`.
 *
 * **والاسمُ من إعدادات المنصّة لا من الشيفرة** (قاعدةُ المالك) — وفارغُه
 * يُسقط القوسين فقط.
 */
fun geoUri(point: Pair<Double, Double>, label: String): String {
    val ll = "${point.first},${point.second}"
    val name = label.trim()
    val q = if (name.isEmpty()) ll else ll + "(" + encodeQuery(name) + ")"
    return "geo:$ll?q=$q"
}

/** **بديلُ من لا تطبيقَ خرائطَ عنده** — خرائط غوغل في المتصفّح. */
fun mapsWebUrl(point: Pair<Double, Double>): String =
    "https://www.google.com/maps/search/?api=1&query=${point.first},${point.second}"

/**
 * **العنوانُ بفواصلَ واحدة** — أسطرٌ وفواصلُ لاتينيّةٌ وعربيّةٌ مختلطةٌ من
 * اللوحة تصير «دمشق، المزة، شارع…» بمسافةٍ واحدة.
 */
fun formatAddress(raw: String): String =
    raw.split('\n', ',', '،')
        .map { it.trim().replace(Regex("\\s+"), " ") }
        .filter { it.isNotEmpty() }
        .joinToString("، ")

/** **ترميزُ نصٍّ داخل الرابط** — `%20` للمسافة لا `+`، فتطبيقُ الخرائط يقرؤه اسما. */
private fun encodeQuery(s: String): String =
    java.net.URLEncoder.encode(s, "UTF-8").replace("+", "%20")
