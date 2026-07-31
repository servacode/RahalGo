"use client";

/**
 * أيقونات تصنيفات المتاجر — مفتاح واحد مخزّن في قاعدة البيانات يقابله أيقونة من
 * مجموعتنا. البديل (الإيموجي) يُرسم بخط النظام فيختلف بين أبل وأندرويد وويندوز،
 * ولا يرث لون التوكنز ولا سماكة الخط، وقد يظهر مربّعاً فارغاً على أنظمة قديمة.
 */

import { getMessages, defaultLocale } from "@rahalgo/i18n";
import {
  IconCatFood,
  IconCatGrocery,
  IconCatPharmacy,
  IconCatSweets,
  IconCatGifts,
  IconCatDrinks,
  IconCatClothes,
  IconCatBakery,
  IconCatButcher,
  IconCatProduce,
  IconCatFlowers,
  IconCatElectronics,
  IconCatBooks,
  IconCatBaby,
  IconCatBeauty,
  IconCatOther,
} from "./icons";

const m = getMessages(defaultLocale);

/** المفاتيح المتاحة — أي إضافة تصنيف جديد تمرّ من هنا لا من حقل نصّي حر. */
export const CATEGORY_ICONS = {
  food: IconCatFood,
  grocery: IconCatGrocery,
  pharmacy: IconCatPharmacy,
  sweets: IconCatSweets,
  gifts: IconCatGifts,
  drinks: IconCatDrinks,
  clothes: IconCatClothes,
  bakery: IconCatBakery,
  butcher: IconCatButcher,
  produce: IconCatProduce,
  flowers: IconCatFlowers,
  electronics: IconCatElectronics,
  books: IconCatBooks,
  baby: IconCatBaby,
  beauty: IconCatBeauty,
  other: IconCatOther,
} as const;

export type CategoryIconKey = keyof typeof CATEGORY_ICONS;

export const CATEGORY_ICON_KEYS = Object.keys(CATEGORY_ICONS) as CategoryIconKey[];

/** اسم عربي للمفتاح — يُستعمل في المنتقي وفي تلميحات الوصول. */
export function categoryIconLabel(key: string): string {
  return (m.terms.categoryIcons as Record<string, string>)[key] ?? m.terms.categoryIcons.other;
}

/** يعرض أيقونة التصنيف من مفتاحه؛ مفتاح مجهول يسقط على "أخرى" بلا كسر. */
export function CategoryIcon({
  name,
  size = 16,
  className = "",
}: {
  name: string | null | undefined;
  size?: number;
  className?: string;
}) {
  const Icon = CATEGORY_ICONS[(name ?? "") as CategoryIconKey] ?? IconCatOther;
  return <Icon size={size} className={className} aria-label={categoryIconLabel(name ?? "other")} />;
}

/** منتقي الأيقونة — بديل حقل النص الحر الذي كان يُكتب فيه إيموجي. */
export function CategoryIconPicker({
  value,
  onChange,
  label,
}: {
  value: string;
  onChange: (key: CategoryIconKey) => void;
  label?: string;
}) {
  return (
    <div>
      {label && <span className="mb-1.5 block text-sm font-medium text-ink">{label}</span>}
      <div className="flex flex-wrap gap-1.5 rounded-control border border-line bg-surface p-2">
        {CATEGORY_ICON_KEYS.map((key) => {
          const Icon = CATEGORY_ICONS[key];
          const active = key === value;
          return (
            <button
              key={key}
              type="button"
              onClick={() => onChange(key)}
              title={categoryIconLabel(key)}
              aria-label={categoryIconLabel(key)}
              aria-pressed={active}
              className={`flex h-9 w-9 items-center justify-center rounded-control border transition-colors ${
                active
                  ? "border-primary bg-primary-light text-primary-dark"
                  : "border-transparent text-ink-muted hover:bg-page hover:text-ink"
              }`}
            >
              <Icon size={18} />
            </button>
          );
        })}
      </div>
    </div>
  );
}
