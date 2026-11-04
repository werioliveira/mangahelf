import { join } from "node:path";
import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  outputFileTracingRoot: join(import.meta.dirname),
};

export default nextConfig;
