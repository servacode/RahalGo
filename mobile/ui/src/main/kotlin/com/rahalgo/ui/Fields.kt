package com.rahalgo.ui

import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.input.PasswordVisualTransformation
import androidx.compose.ui.text.input.VisualTransformation

/**
 * ══════════════════════════════════════════════════════════════════════
 * **حقول متكرّرة — تُكتب مرّة وتُستعمل في كلّ شاشة**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (`GROUND-RULES.md` §2 المركزيّة — والسبب نفسه في الويب.)
 *
 * **وكان حقل الهاتف مكتوبا في شاشتين**، فحملت واحدة أيقونة والأخرى لا،
 * **وحملت واحدة تعليمات الكتابة والأخرى لا** (طلب المالك ٢٠٢٦-٠٨-١١:
 * «أيقونة الهاتف في كلّ حقل رقم، والعين في كلّ مكان فيه كلمة مرور»).
 *
 * **وحقلان متشابهان يفترقان بلا أن ينتبه أحد** — ثمّ يُصلَح أحدهما وحده.
 */

/**
 * حقل رقم الهاتف.
 *
 * **ومثاله داخله**: المحرّك يقبل `0912345678` و`963…` و`+963…`
 * (`identity.NormalizePhone`)، **ويرفض ما عداها برسالة واحدة** لا تقول
 * كيف يُكتب. **فمن كتب رقمه بفواصل أو بمسافات** رأى «رقم غير صحيح» ولا
 * يعرف ما الصحيح.
 */
@Composable
fun PhoneField(
    value: String,
    onChange: (String) -> Unit,
    enabled: Boolean,
    /**
     * **تسميةُ الحقل** — و«رقم الهاتف» في شاشة الدخول.
     *
     * **وشاشةُ الحساب تطلب «الرقم الجديد»** — (قرارُ المالك ٢٠٢٦-٠٨-١٣:
     * «لا تنسَ أيقونة الهاتف لأرقام الهواتف»). **ولو نُسخ الحقلُ هناك
     * ليأخذ تسميةً أخرى** لَفقد أيقونتَه ومثالَه، **وهو عينُ ما تمنعه
     * هذه الملفّة.**
     */
    label: Int = R.string.login_phone,
    /**
     * **يُقرأ ولا يُكتب** — كحقل الرقم المسجَّل في شاشة الحساب.
     *
     * **وغيرُ `enabled = false`**: المعطَّلُ يبهت فيُقرأ «هذا الحقل
     * لا يعنيك»، **والمقروءُ يبقى واضحاً كسائر الحقول** — إنّما لا
     * تُكتب فيه. **وهو رقمُ صاحبه، يجب أن يراه بوضوح.**
     */
    readOnly: Boolean = false,
    modifier: Modifier = Modifier,
) {
    OutlinedTextField(
        value = value,
        onValueChange = onChange,
        readOnly = readOnly,
        label = { Text(stringResource(label)) },
        // **وقالب لا رقم كامل** (`09xxxxxxxx`، طلب المالك ٢٠٢٦-٠٨-١١):
        // **رقم كامل معروض يُقرأ رقما حقيقيّا** — ومن رآه سأل: أهذا رقمي
        // أم رقم الشركة؟ **والقالب يقول الشكل ولا يدّعي أنّه أحد.**
        placeholder = { Text(stringResource(R.string.login_phone_hint)) },
        leadingIcon = { Icon(painterResource(R.drawable.ic_phone), contentDescription = null) },
        singleLine = true,
        // **ولوحة أرقام لا حروف** — رقم الهاتف لا يُكتب بحروف، **ولوحة
        // كاملة تُبطئ من يكتبه كلّ يوم.**
        keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Phone),
        // **الرقمُ من اليسار إلى اليمين دائماً** (الفحصُ البصريّ ٢٠٢٦-١١-١١): في سطرٍ عربيٍّ كانت
        // «+963…» تُرسم «963…+» — **والإشارةُ في آخره.** ويبقى ملاصقاً لأيقونة الهاتف يميناً.
        textStyle = androidx.compose.material3.LocalTextStyle.current.copy(
            textDirection = androidx.compose.ui.text.style.TextDirection.Ltr,
            textAlign = androidx.compose.ui.text.style.TextAlign.Right,
        ),
        enabled = enabled,
        modifier = modifier.fillMaxWidth(),
    )
}

/**
 * حقل كلمة المرور — **وعينه معه أينما كان.**
 *
 * **السائق يكتب على شاشة صغيرة وهو واقف في الشارع**، والنقاط تُخفي غلطة
 * حرف واحد. **فيُقال له «كلمة المرور خطأ» وهي صحيحة** — والغلط في حرف
 * لم يره.
 *
 * **والحالة تبدأ مخفيّة** — من يقف خلفه لا يقرأها إلّا إن أراد هو.
 */
@Composable
fun PasswordField(
    value: String,
    onChange: (String) -> Unit,
    enabled: Boolean,
    label: Int = R.string.login_password,
    modifier: Modifier = Modifier,
    /** **«تم» في الكيبورد يُرسل** (٢٠٢٦-١٠-٠٩) — فلا يلزم إغلاقُه لبلوغ الزرّ. */
    onDone: (() -> Unit)? = null,
) {
    var revealed by remember { mutableStateOf(false) }
    OutlinedTextField(
        value = value,
        onValueChange = onChange,
        label = { Text(stringResource(label)) },
        leadingIcon = { Icon(painterResource(R.drawable.ic_lock), contentDescription = null) },
        trailingIcon = {
            IconButton(onClick = { revealed = !revealed }, enabled = enabled) {
                Icon(
                    painter = painterResource(
                        if (revealed) R.drawable.ic_eye_off else R.drawable.ic_eye,
                    ),
                    contentDescription = stringResource(
                        if (revealed) R.string.login_hide_password
                        else R.string.login_show_password,
                    ),
                )
            }
        },
        singleLine = true,
        visualTransformation =
            if (revealed) VisualTransformation.None else PasswordVisualTransformation(),
        keyboardOptions = KeyboardOptions(
            keyboardType = KeyboardType.Password,
            imeAction = if (onDone != null) androidx.compose.ui.text.input.ImeAction.Go
            else androidx.compose.ui.text.input.ImeAction.Default,
        ),
        keyboardActions = androidx.compose.foundation.text.KeyboardActions(onGo = { onDone?.invoke() }),
        enabled = enabled,
        modifier = modifier.fillMaxWidth(),
    )
}

/** حقل رمز التحقّق — **أرقام لا حروف، ومفتاح يُعرِّفه.** */
@Composable
fun CodeField(
    value: String,
    onChange: (String) -> Unit,
    enabled: Boolean,
    modifier: Modifier = Modifier,
) {
    OutlinedTextField(
        value = value,
        onValueChange = onChange,
        label = { Text(stringResource(R.string.reset_code)) },
        leadingIcon = { Icon(painterResource(R.drawable.ic_key), contentDescription = null) },
        singleLine = true,
        keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.NumberPassword),
        enabled = enabled,
        modifier = modifier.fillMaxWidth(),
    )
}

/**
 * ══════════════════════════════════════════════════════════════════════
 * **الحقلُ الموسوم — كلُّ حقلٍ له اسمٌ يبقى ظاهراً** (نظامُ التصميم ٢٠٢٦-١١-١١، D9)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **حقلٌ بمثالٍ داخله وحدَه يفقد اسمَه حين يُكتب فيه** — فلا يُعرف بعد سطرين ما كان يُطلب. **فالوسمُ
 * إجباريٌّ** (`label`)، والمثالُ اختياريٌّ (`placeholder`)، وسطرُ الشرح أو الخطأ تحته (`supporting`/`error`).
 *
 * ```
 * LabeledField("اسم المتجر", name, { name = it })
 * LabeledField("المبلغ", amount, { amount = it }, keyboardType = KeyboardType.Number, error = amountError)
 * LabeledField("ملاحظة", note, { note = it }, singleLine = false, minLines = 3, supporting = "اختياري")
 * ```
 *
 * @param error نصُّ الخطأ تحت الحقل — وغيرُ الفارغ يلوّن الحقلَ أحمر.
 * @param leadingIcon أيقونةٌ في بداية الحقل (مورد).
 */
@Composable
fun LabeledField(
    label: String,
    value: String,
    onValueChange: (String) -> Unit,
    modifier: Modifier = Modifier,
    placeholder: String = "",
    supporting: String = "",
    error: String = "",
    enabled: Boolean = true,
    readOnly: Boolean = false,
    singleLine: Boolean = true,
    minLines: Int = 1,
    keyboardType: KeyboardType = KeyboardType.Text,
    @androidx.annotation.DrawableRes leadingIcon: Int? = null,
    trailing: (@Composable () -> Unit)? = null,
) {
    OutlinedTextField(
        value = value,
        onValueChange = onValueChange,
        label = { Text(label) },
        placeholder = if (placeholder.isNotEmpty()) ({ Text(placeholder) }) else null,
        supportingText = when {
            error.isNotEmpty() -> ({ Text(error) })
            supporting.isNotEmpty() -> ({ Text(supporting) })
            else -> null
        },
        isError = error.isNotEmpty(),
        leadingIcon = if (leadingIcon != null) ({ Icon(painterResource(leadingIcon), contentDescription = null) }) else null,
        trailingIcon = trailing,
        singleLine = singleLine,
        minLines = if (singleLine) 1 else minLines,
        keyboardOptions = KeyboardOptions(keyboardType = keyboardType),
        enabled = enabled,
        readOnly = readOnly,
        textStyle = com.rahalgo.design.Rahal.type.body,
        modifier = modifier.fillMaxWidth(),
    )
}
