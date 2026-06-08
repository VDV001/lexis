import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  // Emit a self-contained server bundle (.next/standalone) for a small
  // production Docker image. See lexis-web/Dockerfile.
  output: "standalone",
};

export default nextConfig;
