import type { Config } from "tailwindcss";
import { heroui } from "@heroui/react";

const config: Config = {
  content: [
    "./src/pages/**/*.{js,ts,jsx,tsx,mdx}",
    "./src/components/**/*.{js,ts,jsx,tsx,mdx}",
    "./src/app/**/*.{js,ts,jsx,tsx,mdx}",
    "./node_modules/@heroui/theme/dist/**/*.{js,ts,jsx,tsx}",
  ],
  theme: {
    extend: {},
  },
  darkMode: "class",
  plugins: [
    heroui({
      themes: {
        light: {
          colors: {
            background: "#f8fafc",
            foreground: "#1e293b",
            primary: {
              50: "#ecfeff",
              100: "#cffafe",
              200: "#a5f3fc",
              300: "#67e8f9",
              400: "#22d3ee",
              500: "#06b6d4",
              600: "#0891b2",
              700: "#0e7490",
              800: "#155e75",
              900: "#164e63",
              DEFAULT: "#06b6d4",
              foreground: "#ffffff",
            },
          },
        },
        dark: {
          colors: {
            background: "#0f1729",
            foreground: "#e2e8f0",
            primary: {
              50: "#ecfeff",
              100: "#cffafe",
              200: "#a5f3fc",
              300: "#67e8f9",
              400: "#22d3ee",
              500: "#06b6d4",
              600: "#0891b2",
              700: "#0e7490",
              800: "#155e75",
              900: "#164e63",
              DEFAULT: "#22d3ee",
              foreground: "#000000",
            },
          },
        },
      },
    }),
  ],
};

export default config;
