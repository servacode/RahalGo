/**
 * **بديلُ حزمةِ الواجهة في مِسند الاختبار.**
 *
 * **وحزمةُ الواجهة فيها JSX لا ينزعه `node`** — **وليست هي المفحوسة.**
 * **والمفحوصُ عميلُ الـAPI بشيفرته الحقيقيّة**، وكلُّ ما يأخذه منها
 * عنوانُ المحرّك. **فيُقدَّم هذا وحدَه، ولا يُمَسّ المفحوص.**
 *
 * **وثمنُ البديل معروف**: لو تبدّل عقدُ `apiBase` لم يكشفه هذا المِسند
 * — **ويكشفه `tsc` وحرّاسُ التهيئة**، وهما يمرّان على المصدر نفسِه.
 */
export const apiBase = () => globalThis.__RAHALGO_CONFIG__?.apiUrl ?? "";
export const wsBase = () => "";
export const runtimeConfig = () => globalThis.__RAHALGO_CONFIG__ ?? {};
export const isStagingEnv = () => false;
export const mapStyleUrl = () => "";
export const mapTilesUrl = () => "";
