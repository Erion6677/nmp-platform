"use client";

import { HeroUIProvider } from "@heroui/react";
import { ThemeProvider as NextThemesProvider } from "next-themes";
import { useRouter } from "next/navigation";
import AuthGuard from "@/components/auth/AuthGuard";

export function Providers({ children }: { children: React.ReactNode }) {
  const router = useRouter();

  return (
    <NextThemesProvider attribute="class" defaultTheme="dark" enableSystem>
      <HeroUIProvider navigate={router.push}>
        <AuthGuard>{children}</AuthGuard>
      </HeroUIProvider>
    </NextThemesProvider>
  );
}
