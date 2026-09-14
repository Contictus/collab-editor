/** @type {import('next').NextConfig} */
const nextConfig = {
  reactStrictMode: true,
  // Workspace packages ship TS source; let Next transpile them.
  transpilePackages: ['db', 'protocol', 'shared'],
};

export default nextConfig;
