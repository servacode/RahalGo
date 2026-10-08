package com.rahalgo.ui

import android.content.Intent
import android.net.Uri
import androidx.activity.compose.BackHandler
import androidx.compose.animation.core.RepeatMode
import androidx.compose.animation.core.animateFloat
import androidx.compose.animation.core.infiniteRepeatable
import androidx.compose.animation.core.rememberInfiniteTransition
import androidx.compose.animation.core.tween
import androidx.compose.foundation.Image
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.setValue
import kotlinx.coroutines.launch
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.graphicsLayer
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.unit.dp
import com.rahalgo.design.Rahal

/**
 * ══════════════════════════════════════════════════════════════════════
 * **شاشةُ التحديث — لا تُغلق ولا يُرجع منها**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (طلبُ المالك ٢٠٢٦-٠٨-٢٥: «التطبيق يجب أن تظهر شاشةٌ تجبر المستخدمَ
 *  على تحديث التطبيق من غوغل بلاي أو من الرابط الخارجيّ — مو معقول
 *  التطبيق يتحدّث والمستخدم ما عنده خبر بالشي».)
 *
 * # ولماذا لا تُغلق
 *
 * **ونسخةٌ تخلّفت عن المحرّك تنكسر عند صاحبها بصمت**: نداءٌ يردّ حقلاً
 * لا تعرفه، وشاشةٌ تُرسم بلا بيانات. **ورسالةٌ تُغلق بضغطةٍ تُغلق
 * وتُنسى.**
 *
 * **والرجوعُ يخرج من التطبيق لا يُدخِله** (تصحيحُ المالك ٢٠٢٦-٠٩-٢٦) —
 * **فلا يبقى صاحبُه عالقاً على الشاشة بزرٍّ ميّت، ولا يتسلّل إلى التطبيق
 * دون تحديثٍ فيظنّه اختياريّاً.** يُغلَق التطبيقُ، وعند فتحِه ثانيةً تعود
 * الشاشةُ ما دامت النسخةُ قديمة.
 *
 * # وزرّان صريحان لا زرٌّ واحد
 *
 * (قرارُ المالك ٢٠٢٦-٠٩-٢٦: «زرٌّ خاصٌّ بغوغل بلاي وزرٌّ للتحديث المباشر».)
 *
 * **زرُّ Google Play** — لمن نزّل منه (وإن لم يكن بلاي على الجهاز فُتحت
 * صفحتُه على الويب). **وزرُّ التنزيل المباشر** — للأجهزة بلا خدمات غوغل
 * (كثيرةٌ في سوريا)، يفتح رابطَ المتجر المباشر. **والاختيارُ صريحٌ للمستخدم**
 * لا محاولةٌ صامتةٌ يظنّ معها أنّ التحديثَ تعذّر.
 */
// **وقناةُ التوزيع تفرّق الأزرار** (٢٠٢٦-٠٩-٢٦):
//
// **الزبونُ على Google Play**، فله زرُّ المتجر أوّلاً والمباشرُ احتياطاً.
// **والمندوبُ (وأخواه لاحقاً) توزيعٌ مباشرٌ من الموقع لا غير** — فلا زرَّ Play
// (رابطُه إلى صفحةٍ لا وجودَ لها فيُوهم أنّ التحديثَ متعذّر)، بل زرُّ التنزيل
// المباشر وحدَه رئيسيّاً إلى رابطِ تطبيقه بعينه. **والنصُّ يُمرَّر ملائماً
// لدور التطبيق** (المندوبُ لا «يطلب»)، وإلّا فالنصُّ العامّ.
@Composable
fun UpdateGate(
    pkg: String,
    fallbackUrl: String = "https://rahalgo.com/app",
    showPlay: Boolean = true,
    body: String? = null,
    /**
     * **مفتاحُ التطبيق للتحديث من داخله** (`driver`/`merchant`/`rep`) — يُنزَّل
     * الملفُّ هنا وتُفتح شاشةُ «تثبيت» (`SelfUpdate`). وفارغٌ: بلاي أو الرابط.
     */
    selfUpdateKey: String? = null,
) {
    val ctx = LocalContext.current
    val scope = androidx.compose.runtime.rememberCoroutineScope()
    var progress by androidx.compose.runtime.remember { androidx.compose.runtime.mutableStateOf<Float?>(null) }
    var ready by androidx.compose.runtime.remember { androidx.compose.runtime.mutableStateOf<java.io.File?>(null) }
    var failed by androidx.compose.runtime.remember { androidx.compose.runtime.mutableStateOf(false) }

    // **والرجوعُ يخرج من التطبيق** — انظر أعلاه: لا بقاءَ عالقاً، ولا تسلّلَ للداخل.
    BackHandler(enabled = true) { (ctx as? android.app.Activity)?.finish() }

    // **ونبضُ الشعار واضحٌ** (قرار المالك ٢٠٢٦-٠٩-٢٦) — يكبر ويصغر ببطءٍ ملحوظ
    // فيلفت النظرَ إلى أنّ ثمّة تحديثاً مطلوباً.
    val pulse = rememberInfiniteTransition(label = "updateLogo")
    val logoScale by pulse.animateFloat(
        initialValue = 1f,
        targetValue = 1.12f,
        animationSpec = infiniteRepeatable(tween(700), RepeatMode.Reverse),
        label = "updateLogoScale",
    )

    Surface(Modifier.fillMaxSize(), color = Rahal.colors.canvas) {
        Column(
            modifier = Modifier.fillMaxSize().padding(horizontal = 32.dp),
            horizontalAlignment = Alignment.CenterHorizontally,
            verticalArrangement = Arrangement.Center,
        ) {
            // **شعارُ رحّال غو أعلى الصفحة** — الرسميُّ نفسُه (`intro_logo`)، ينبض.
            Image(
                painter = painterResource(com.rahalgo.design.R.drawable.intro_logo),
                contentDescription = null,
                modifier = Modifier
                    .size(140.dp)
                    .graphicsLayer { scaleX = logoScale; scaleY = logoScale },
            )
            Spacer(Modifier.height(24.dp))
            Text(
                text = stringResource(R.string.update_title),
                color = Rahal.colors.ink,
                style = MaterialTheme.typography.headlineSmall,
                textAlign = TextAlign.Center,
            )
            Spacer(Modifier.height(12.dp))
            Text(
                text = body ?: stringResource(R.string.update_body),
                color = Rahal.colors.inkMuted,
                style = MaterialTheme.typography.bodyMedium,
                textAlign = TextAlign.Center,
            )
            Spacer(Modifier.height(28.dp))
            val openDirect = {
                runCatching {
                    ctx.startActivity(
                        Intent(Intent.ACTION_VIEW, Uri.parse(fallbackUrl))
                            .addFlags(Intent.FLAG_ACTIVITY_NEW_TASK),
                    )
                }
                Unit
            }
            // ── التحديثُ من داخل التطبيق (التوزيعُ المباشر) ──
            if (selfUpdateKey != null) {
                val installOrAsk: (java.io.File) -> Unit = { f ->
                    if (SelfUpdate.canInstall(ctx)) {
                        SelfUpdate.install(ctx, f)
                    } else {
                        SelfUpdate.openInstallPermission(ctx)
                    }
                }
                val p = progress
                val file = ready
                when {
                    file != null -> {
                        RahalButton(onClick = { installOrAsk(file) }, modifier = Modifier.fillMaxWidth()) {
                            Text(
                                stringResource(
                                    if (SelfUpdate.canInstall(ctx)) R.string.update_install else R.string.update_allow,
                                ),
                            )
                        }
                        Spacer(Modifier.height(8.dp))
                        Text(
                            stringResource(R.string.update_install_hint),
                            color = Rahal.colors.inkMuted,
                            style = MaterialTheme.typography.bodySmall,
                            textAlign = TextAlign.Center,
                        )
                    }
                    p != null -> {
                        androidx.compose.material3.LinearProgressIndicator(
                            progress = { p },
                            modifier = Modifier.fillMaxWidth(),
                        )
                        Spacer(Modifier.height(8.dp))
                        Text(
                            stringResource(R.string.update_downloading, (p * 100).toInt()),
                            color = Rahal.colors.inkMuted,
                            style = MaterialTheme.typography.bodyMedium,
                        )
                    }
                    else -> {
                        RahalButton(
                            onClick = {
                                failed = false
                                progress = 0f
                                scope.launch {
                                    runCatching {
                                        SelfUpdate.download(ctx, SelfUpdate.apkUrl(selfUpdateKey)) { progress = it }
                                    }.onSuccess { f ->
                                        ready = f
                                        progress = null
                                        installOrAsk(f)
                                    }.onFailure {
                                        progress = null
                                        failed = true
                                    }
                                }
                            },
                            modifier = Modifier.fillMaxWidth(),
                        ) { Text(stringResource(R.string.update_action)) }
                        if (failed) {
                            Spacer(Modifier.height(8.dp))
                            Text(
                                stringResource(R.string.update_failed),
                                color = Rahal.colors.danger,
                                style = MaterialTheme.typography.bodySmall,
                                textAlign = TextAlign.Center,
                            )
                        }
                    }
                }
                Spacer(Modifier.height(12.dp))
                RahalTextButton(onClick = openDirect) { Text(stringResource(R.string.update_direct)) }
            } else if (showPlay) {
                // ── بلاي: التحديثُ داخل التطبيق، والمتجرُ احتياط ──
                val openMarket = {
                    val market = Intent(
                        Intent.ACTION_VIEW,
                        Uri.parse("market://details?id=$pkg"),
                    ).addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)
                    val ok = runCatching { ctx.startActivity(market) }.isSuccess
                    if (!ok) {
                        runCatching {
                            ctx.startActivity(
                                Intent(
                                    Intent.ACTION_VIEW,
                                    Uri.parse("https://play.google.com/store/apps/details?id=$pkg"),
                                ).addFlags(Intent.FLAG_ACTIVITY_NEW_TASK),
                            )
                        }
                    }
                    Unit
                }
                RahalButton(
                    onClick = {
                        val activity = ctx as? android.app.Activity
                        if (activity == null) {
                            openMarket()
                        } else {
                            PlayUpdate.start(activity, onUnavailable = openMarket)
                        }
                    },
                    modifier = Modifier.fillMaxWidth(),
                ) { Text(stringResource(R.string.update_action)) }
                Spacer(Modifier.height(12.dp))
                RahalOutlineButton(onClick = openDirect, modifier = Modifier.fillMaxWidth()) {
                    Text(stringResource(R.string.update_direct))
                }
            } else {
                RahalButton(onClick = openDirect, modifier = Modifier.fillMaxWidth()) {
                    Text(stringResource(R.string.update_direct))
                }
            }
        }
    }
}
