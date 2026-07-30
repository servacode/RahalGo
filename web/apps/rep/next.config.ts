import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  transpilePackages: ["@rahalgo/ui", "@rahalgo/i18n"],
  env: {
    NEXT_PUBLIC_API_URL: process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080",
  },
};

export default nextConfig;
