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
        keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Password),
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
