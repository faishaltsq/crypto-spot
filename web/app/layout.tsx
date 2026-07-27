import type { Metadata } from "next";
import { Providers } from "@/components/providers";

import "./globals.css";

export const metadata: Metadata = {
  title: "Crypto Spot Signal",
  description: "Gate SPOT realtime paper signal dashboard",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en" suppressHydrationWarning>
      <body>
        <Providers>
          {children}
        </Providers>
      </body>
    </html>
  );
}
