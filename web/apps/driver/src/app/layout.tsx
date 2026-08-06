import type { Metadata, Viewport } from "next";
import { getMessages, getDir, defaultLocale } from "@rahalgo/i18n";
import { PlatformProvider } from "@rahalgo/ui";
import { AuthProvider } from "@/lib/auth";
// خط المنصة — مصدر مركزي واحد (packages/ui/src/fonts.css)
import "@rahalgo/ui/fonts.css";
import "./globals.css";

const m = getMessages(defaultLocale);

export const metadata: Metadata = {
  title: m.driver.appTitle,
  description: m.driver.appDescription,
};

/**
 * هذا التطبيق وحده يمنع التكبير بالقرص.
 *
 * البقية تُقرأ على مكتب، وهذا يُقرأ بيدٍ واحدة والسائق واقفٌ عند درّاجته: قرصةٌ
 * عرَضية تُزيح الزرّ الكبير عن مكانه فيضغط ما لم يُرد. وسيصير تطبيق أندرويد
 * لاحقاً، فالسلوك هنا يُحاكي ما سيكون عليه هناك.
 */
export const viewport: Viewport = {
  width: "device-width",
  initialScale: 1,
  maximumScale: 1,
  userScalable: false,
};

/** أصلُ المحرّك — **هويّةُ المنصة تُقرأ منه لا من المعجم.** */
const API = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang={defaultLocale} dir={getDir(defaultLocale)}>
      <body className="flex min-h-screen flex-col">
        <PlatformProvider apiBase={API}>
          <AuthProvider>{children}</AuthProvider>
        </PlatformProvider>
      </body>
    </html>
  );
}
