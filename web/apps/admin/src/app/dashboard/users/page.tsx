"use client";

/**
 * **«الحسابات» — بابٌ واحدٌ لكلّ من في المنصة.**
 *
 * # لماذا اجتمعت
 *
 * كانت خمسةَ أبوابٍ للشيء الواحد: «الحسابات» فيها الجميع، **و«الزبائن»
 * و«السائقون» و«المندوبون» و«المتاجر» كلٌّ يعيد عرضَ فئةٍ منهم.** فمن أراد أن
 * يعرف إنساناً **فتح بابين ولم يجد صورتَه في أيّهما كاملة**، ومن أراد أن يبحث
 * لم يعرف في أيّ بابٍ يبحث.
 *
 * قرارُ المالك (٢٠٢٦-٠٨-٠٣): «يمكننا حذفُ أقسام المندوب والسائقين والزبائن
 * والمتاجر بما أنّها مجموعةٌ كلُّها بقسم الحسابات، فلا داعي أن تكون بمكانين».
 *
 * # ولا ميزةَ تُخسَر — **وهذا شرطُه**
 *
 * «**ولا نريد خسارةَ أيّ ميزة عند حذف الأقسام المنفصلة**».
 *
 * وفي كلٍّ منها **أعمدةٌ لا وجودَ لها في جدول الحسابات**: دوامُ السائق ونقدُه
 * وطلباتُه المفتوحة · إنفاقُ الزبون وآخرُ طلبٍ له · كودُ المندوب ومتاجرُه
 * وعمولاتُه · وقائمةُ المتجر وساعاتُه ومخالفاتُه.
 *
 * **فلم تُحذف الشاشاتُ — نُقلت.** كلُّ واحدةٍ صارت مكوّناً في `components/accounts`
 * **بجدولها وأزرارها ونداءاتها كما هي**، وهذه الصفحةُ تعرضها في تبويبها.
 * **وحذفُ ملفٍّ وإعادةُ كتابته من الذاكرة هو ما يُضيّع الأعمدة** — والنقلُ لا
 * يُضيّع شيئاً.
 *
 * # والمتجرُ ليس إنساناً
 *
 * **وهو الاستثناءُ الذي يستحقّ أن يُقال**: الزبونُ والسائقُ والمندوب **حسابات**،
 * والمتجرُ **كيانٌ** له قائمةٌ وساعاتٌ وعمولةٌ ومخالفات — **وصاحبُه حسابٌ آخر.**
 * فيبقى في تبويبه هنا للوصول، **ولا يُدمج في جدول الأشخاص** لأنّ أعمدتَه ليست
 * أعمدتَهم.
 */

import { useState } from "react";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import { IconUsers, IconUser, IconStore, IconDriver, IconLink } from "@rahalgo/ui";
import AllAccountsTable from "@/components/accounts/all";
import CustomersTable from "@/components/accounts/customers";
import DriversTable from "@/components/accounts/drivers";
import SalesTable from "@/components/accounts/sales";
import MerchantsTable from "@/components/accounts/merchants";

const m = getMessages(defaultLocale);
const T = m.admin.users.groupTabs;

type Tab = "all" | "customers" | "merchants" | "drivers" | "reps";

const TABS: { key: Tab; label: string; icon: React.ReactNode }[] = [
  { key: "all", label: T.all, icon: <IconUsers size={15} /> },
  { key: "customers", label: T.customers, icon: <IconUser size={15} /> },
  { key: "merchants", label: T.merchants, icon: <IconStore size={15} /> },
  { key: "drivers", label: T.drivers, icon: <IconDriver size={15} /> },
  { key: "reps", label: T.reps, icon: <IconLink size={15} /> },
];

export default function AccountsPage() {
  const [tab, setTab] = useState<Tab>("all");

  return (
    <div>
      <div className="mb-4 flex flex-wrap gap-1 border-b border-line">
        {TABS.map((t) => (
          <button
            key={t.key}
            onClick={() => setTab(t.key)}
            className={`flex items-center gap-1.5 border-b-2 px-4 py-2 text-sm transition-colors ${
              tab === t.key
                ? "border-primary font-bold text-primary-dark"
                : "border-transparent text-ink-muted hover:text-ink"
            }`}
          >
            {t.icon}
            {t.label}
          </button>
        ))}
      </div>

      {/* **ولا يُحمَّل تبويبٌ لم يُفتح** — خمسةُ جداولَ تُنادى معاً حملٌ بلا حاجة،
          **ومن يريد قائمةَ الزبائن لا ينتظر قائمةَ المتاجر.** */}
      {tab === "all" && <AllAccountsTable />}
      {tab === "customers" && <CustomersTable />}
      {tab === "merchants" && <MerchantsTable />}
      {tab === "drivers" && <DriversTable />}
      {tab === "reps" && <SalesTable />}
    </div>
  );
}
