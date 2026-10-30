/**
 * **ترتيبُ الفاتورة: الأصنافُ أوّلاً ثمّ المشروباتُ كلُّها تحتها** (قرارُ المالك ٢٠٢٦-١٠-٣٠)
 *
 * تُنادى `invoiceOrder` نفسُها (`packages/ui/src/invoiceText.ts`) — وهي ما تقرؤه فاتورةُ الزبون
 * واللوحةُ ولوحةُ المتجر — **ويُقاس الناتج**: المشروبُ نفسُه مع صنفين سطرٌ واحدٌ بعددٍ ومجموعٍ مجموعين.
 */
import { join, dirname } from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";
import { register } from "node:module";

const here = dirname(fileURLToPath(import.meta.url));
register(pathToFileURL(join(here, "_tsload.mjs")).href);

const { invoiceOrder } = await import(pathToFileURL(join(here, "..", "packages", "ui", "src", "invoiceText.ts")).href);

let fails = 0;
const eq = (name, got, want) => {
  const g = JSON.stringify(got);
  const w = JSON.stringify(want);
  if (g !== w) {
    fails++;
    console.error(`✗ ${name}\n   got:  ${g}\n   want: ${w}`);
  }
};

// ١ · صنفان، ومع كلٍّ منهما «كولا»، ومع الثاني «عصير» — والمحرّكُ يرسلها كلَّ صنفٍ ثمّ مشروباته.
const lines = [
  { id: "a", name: "شاورما", qty: 2, unit_price: 30 },
  { id: "a1", name: "كولا", qty: 1, unit_price: 10, drink_for: "a" },
  { id: "b", name: "فلافل", qty: 1, unit_price: 20 },
  { id: "b1", name: "كولا", qty: 2, unit_price: 10, drink_for: "b" },
  { id: "b2", name: "عصير", qty: 1, unit_price: 15, drink_for: "b" },
];
const out = invoiceOrder(lines);
eq("الأصنافُ أوّلاً ثمّ المشروبات", out.map((l) => l.name), ["شاورما", "فلافل", "كولا", "عصير"]);
eq("الكولا سطرٌ واحدٌ بعدد ٣", out[2].qty, 3);
eq("ومجموعُها ٣٠", out[2].unit_price * out[2].qty, 30);
eq("ومعرّفاتُ أسطرها", out[2].merged_ids, ["a1", "b1"]);
eq("والعصيرُ كما هو", out[3], lines[4]);

// ٢ · مشروبٌ بسعرين مختلفين — المجموعُ يبقى صحيحاً.
const mixed = invoiceOrder([
  { id: "a", name: "برغر", qty: 1, unit_price: 50 },
  { id: "a1", name: "ماء", qty: 1, unit_price: 5, drink_for: "a" },
  { id: "b", name: "برغر دبل", qty: 1, unit_price: 70 },
  { id: "b1", name: "ماء", qty: 3, unit_price: 6, drink_for: "b" },
]);
eq("سعران مختلفان: العدد ٤", mixed[2].qty, 4);
eq("سعران مختلفان: المجموع ٢٣", Math.round(mixed[2].unit_price * mixed[2].qty), 23);

// ٣ · طلبٌ قديمٌ بلا `drink_for` لا يتغيّر.
const old = [
  { id: "x", name: "شاورما", qty: 1, unit_price: 30, options: [{ name: "كولا" }] },
  { id: "y", name: "فلافل", qty: 1, unit_price: 20 },
];
eq("الطلبُ القديمُ كما هو", invoiceOrder(old), old);

// ٤ · مشروبٌ صنفُه ليس في القائمة لا يضيع.
eq(
  "مشروبٌ يتيمٌ يبقى",
  invoiceOrder([{ id: "z1", name: "كولا", qty: 1, unit_price: 10, drink_for: "gone" }]).map((l) => l.id),
  ["z1"],
);

if (fails > 0) {
  console.error(`✗ ترتيبُ الفاتورة: ${fails} فحصاً سقط`);
  process.exit(1);
}
console.log("✓ ترتيبُ الفاتورة: الأصنافُ ثمّ المشروباتُ مدموجة");
