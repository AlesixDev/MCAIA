import type { Metadata } from "next";
import { JetBrains_Mono, Onest } from "next/font/google";

import { SessionProvider } from "@/lib/session-context";

import "./globals.css";

const onest = Onest({
  variable: "--font-onest",
  subsets: ["latin"],
});

const mono = JetBrains_Mono({
  variable: "--font-mono-code",
  subsets: ["latin"],
});

export const metadata: Metadata = {
  title: "mcaia",
  description: "Minecraft animations with local AI",
};

export default function RootLayout({ children }: LayoutProps<"/">) {
  return (
    <html lang="en" className={`${onest.variable} ${mono.variable} dark h-full`}>
      <body className="min-h-full">
        <SessionProvider>{children}</SessionProvider>
      </body>
    </html>
  );
}
