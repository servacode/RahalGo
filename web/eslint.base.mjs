// ══════════════════════════════════════════════════════════════════════
// **قواعدُ الخطّافات — وسببُ وجودها حادثةٌ بعينها**
// ══════════════════════════════════════════════════════════════════════
//
// (٢٠٢٦-٠٩-٠٥: شاشةُ الإعدادات سقطت في الإنتاج بـ«Rendered more hooks
//  than during the previous render» — **وُضع `useMemo` تحت خروجٍ مبكّر**.)
//
// **ولم يمسكه شيء**: `tsc` رآه سليماً، وNext بنته بنجاح، والحاويةُ
// أقلعت `✓ Ready`، وسجلُّ الخادم نظيفٌ تماماً — **لأنّ العطبَ في
// المتصفّح وحدَه.** ولم يظهر إلّا حين فتحها المالك.
//
// **و`pnpm lint` كان يطبع «No tasks were executed» ويُقرأ نجاحاً** —
// فلم يكن في المشروع `eslint` أصلاً: لا حزمةٌ ولا سكربت.
//
// # وما لا يُضاف هنا
//
// **لا `eslint-config-next` ولا قواعدُ أسلوب.** المطلوبُ حارسٌ يمسك
// صنفَ العطب الذي وقع، **وقاعدةُ أسلوبٍ في مشروعٍ قائمٍ تُنتج ألفَ
// تحذيرٍ يُعلّم العينَ أن تتخطّى المخرجاتِ كلَّها** — فيصير الحارسُ
// ضجيجاً لا حارساً.

import js from "@eslint/js";
import tseslint from "typescript-eslint";
import reactHooks from "eslint-plugin-react-hooks";
import next from "@next/eslint-plugin-next";

export default tseslint.config(
  {
    ignores: [
      "**/.next/**",
      "**/dist/**",
      "**/node_modules/**",
      "**/public/**",
      "**/*.config.mjs",
      "**/*.config.ts",
      "**/next-env.d.ts",
    ],
  },
  js.configs.recommended,
  ...tseslint.configs.recommended,
  {
    files: ["**/*.{ts,tsx}"],
    plugins: { "react-hooks": reactHooks, "@next/next": next },
    rules: {
      // ══════════════════════════════════════════════════════════════
      // **الحارسُ المقصود — والخطأُ لا التحذير**
      // ══════════════════════════════════════════════════════════════
      //
      // **`rules-of-hooks` هي التي تمسك حادثةَ ٢٠٢٦-٠٩-٠٥ بعينها**:
      // خطّافٌ بعد `return` مشروط.
      //
      // **و`exhaustive-deps` تحذيرٌ لا خطأ** — فهي تُنتج إيجابيّاتٍ
      // كاذبةً في شيفرةٍ قائمة، **وإسقاطُ البناء بها يُغري بتعطيل
      // الحارس كلِّه.** والتحذيرُ يُقرأ ويُصلَح على مهل.
      "react-hooks/rules-of-hooks": "error",

      /* **وملحقُ Next يُسجَّل ليُعرَف اسمُ قاعدته لا لتُفرَض.**

         **الشيفرةُ فيها `eslint-disable-next-line @next/next/no-img-element`
         في ثمانية مواضع** — وُضعت عن قصد. **وESLint 9 يعدّ تعطيلَ قاعدةٍ
         مجهولةٍ خطأً**، فيسقط الفحصُ على تعليقاتٍ صحيحة.

         **والقاعدةُ مُطفأةٌ لا مُشعَلة**: `<img>` مقصودٌ في تلك المواضع،
         **وإشعالُها اليومَ يُنتج ثمانيةَ أخطاءٍ لا علاقةَ لها بما وقع.** */
      "@next/next/no-img-element": "off",
      "react-hooks/exhaustive-deps": "warn",

      // **وما دون ذلك يُخفَّض** — الحارسُ لأجل الخطّافات، **وشيفرةٌ
      // قائمةٌ تُقاس بقواعدَ لم تُكتب لها تُنتج ضجيجاً يُخفي الإشارة.**
      "@typescript-eslint/no-explicit-any": "off",
      "@typescript-eslint/no-unused-vars": [
        "warn",
        { argsIgnorePattern: "^_", varsIgnorePattern: "^_" },
      ],
      "@typescript-eslint/no-empty-object-type": "off",
      "no-empty": ["warn", { allowEmptyCatch: true }],
    },
  },
);
