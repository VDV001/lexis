import type { Metadata } from "next";
import "./globals.css";

export const metadata: Metadata = {
  title: "lang.tutor",
  description: "AI-powered language learning platform",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="ru" className="h-full">
      <body className="h-screen flex flex-col">{children}</body>
    </html>
  );
}
