import type { Metadata } from "next";
import { getMessages, getDir, defaultLocale } from "@rahalgo/i18n";
import { AuthProvider } from "@/lib/auth";
// خط المنصة — مصدر مركزي واحد (packages/ui/src/fonts.css)
import "@rahalgo/ui/fonts.css";
import "./globals.css";

const m = getMessages(defaultLocale);

export const metadata: Metadata = {
  title: m.rep.appTitle,
  description: m.rep.appDescription,
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang={defaultLocale} dir={getDir(defaultLocale)}>
      <body className="flex min-h-screen flex-col">
        <AuthProvider>{children}</AuthProvider>
      </body>
    </html>
  );
}
