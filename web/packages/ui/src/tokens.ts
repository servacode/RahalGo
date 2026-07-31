/**
 * توكنز التصميم المركزية — المصدر الوحيد للقيم البصرية في كل الواجهات.
 * القيم معتمدة من docs/BRAND.md — أي تعديل يبدأ هناك ثم يُعكس هنا وفي theme.css.
 * ممنوع استخدام قيم مباشرة في أي مكوّن (GROUND-RULES §1.2).
 */
export const colors = {
  brand: {
    primary: "#0E7490",
    primaryDark: "#155E75",
    primaryLight: "#ECFEFF",
    accent: "#F59E0B",
    accentDark: "#D97706",
  },
  status: {
    success: "#16A34A",
    warning: "#EAB308",
    danger: "#DC2626",
    info: "#0284C7",
    violet: "#7C3AED",
  },
  neutral: {
    textPrimary: "#1E293B",
    textSecondary: "#64748B",
    border: "#E2E8F0",
    surface: "#FFFFFF",
    page: "#F8FAFC",
  },
} as const;

export const fontFamily = {
  /** الخط المعتمد للعربية واللاتينية معاً (BRAND.md) */
  sans: `"Tajawal", "IBM Plex Sans Arabic", system-ui, sans-serif`,
} as const;

export const radius = {
  control: "8px",
  card: "16px",
  badge: "9999px",
} as const;

/** نقاط القياس الموحدة للتجاوب (GROUND-RULES §2) */
export const breakpoints = {
  sm: "640px",
  md: "768px",
  lg: "1024px",
  xl: "1280px",
} as const;
