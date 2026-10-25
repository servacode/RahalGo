/**
 * **شارةُ حال الصنف فوق صورته — لونٌ مصمَتٌ ونصٌّ أبيض** (قرارُ المالك ٢٠٢٦-١٠-٢٥: «كلمة معروض فوق
 * الصورة مو مبيّنة»). الشارةُ الفاتحة تذوب فوق صورةٍ فاتحة، **والمصمَتةُ تُقرأ فوق أيّ صورة.**
 */
const SOLID = {
  success: "bg-success-solid",
  warning: "bg-warning-solid",
  danger: "bg-danger-solid",
} as const;

export default function StateBadge({
  label,
  variant,
}: {
  label: string;
  variant: keyof typeof SOLID;
}) {
  return (
    <span className={`inline-flex items-center rounded-full px-2 py-0.5 text-2xs font-bold text-on-solid elev-1 ${SOLID[variant]}`}>
      {label}
    </span>
  );
}
