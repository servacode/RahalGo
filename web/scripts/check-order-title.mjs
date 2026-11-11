/**
 * **عنوانُ الطلب: «+N» أصنافٌ أخرى لا قطع** (قرارُ المالك ٢٠٢٦-١١-١١)
 *
 * صنفٌ واحدٌ بكميّة ٢ ← «برغر كلاسيك × ٢»، وصنفان مختلفان ← «برغر كلاسيك +١». توأمُه في التطبيق `ItemsTitleTest`.
 */
import { join, dirname } from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";
import { register } from "node:module";

const here = dirname(fileURLToPath(import.meta.url));
register(pathToFileURL(join(here, "_tsload.mjs")).href);
const { itemsTitleParts, itemsTitleText } = await import(
  pathToFileURL(join(here, "..", "apps", "rahalgo", "src", "lib", "customer", "orderTitle.ts")).href
);

let fails = 0;
const eq = (name, got, want) => {
  if (got !== want) {
    fails++;
    console.error(`✗ ${name}\n   got:  ${got}\n   want: ${want}`);
  }
};
const fmt = (n) => String(n);
const title = (items, preview = "") => itemsTitleText(itemsTitleParts(items, preview, "،"), fmt);

eq("سطرٌ واحدٌ بكميّة ٢", title([{ name: "برغر كلاسيك", qty: 2, menu_item_id: "a" }]), "برغر كلاسيك × 2");
eq("سطرٌ واحدٌ بكميّة ١", title([{ name: "برغر كلاسيك", qty: 1, menu_item_id: "a" }]), "برغر كلاسيك");
eq(
  "صنفان مختلفان",
  title([
    { name: "برغر كلاسيك", qty: 2, menu_item_id: "a" },
    { name: "بطاطا", qty: 1, menu_item_id: "b" },
  ]),
  "برغر كلاسيك +1",
);
eq(
  "الصنفُ نفسُه بخيارين سطران لكنّه صنفٌ واحد",
  title([
    { name: "برغر كلاسيك", qty: 1, menu_item_id: "a" },
    { name: "برغر كلاسيك", qty: 1, menu_item_id: "a" },
  ]),
  "برغر كلاسيك × 2",
);
eq(
  "المشروبُ ليس صنفاً",
  title([
    { name: "برغر كلاسيك", qty: 1, menu_item_id: "a" },
    { name: "كولا", qty: 2, drink_for: "x" },
  ]),
  "برغر كلاسيك",
);
eq("بلا بنودٍ: المعاينة", title(null, "شاورما، فلافل، كولا"), "شاورما +2");
eq("لا شيء", title(null, ""), "");

if (fails) process.exit(1);
console.log("check-order-title: OK");
