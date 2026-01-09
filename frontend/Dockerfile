# 使用 Node.js 22.x Debian-slim 版本（避免 Alpine SIGSEGV 问题）
FROM node:22-slim AS base

# 安装依赖阶段
FROM base AS deps
WORKDIR /app

# 安装构建依赖（Debian 版本）
RUN apt-get update && apt-get install -y --no-install-recommends \
    python3 make g++ && rm -rf /var/lib/apt/lists/*

# 复制 package 文件
COPY package.json package-lock.json* ./

# 安装依赖
RUN npm ci --legacy-peer-deps

# 构建阶段
FROM base AS builder
WORKDIR /app

# 复制依赖
COPY --from=deps /app/node_modules ./node_modules
COPY . .

# 设置环境变量
ENV NEXT_TELEMETRY_DISABLED=1
ENV NODE_ENV=production
ENV NODE_OPTIONS="--max-old-space-size=2048"

# 创建生产环境配置
RUN echo "NEXT_PUBLIC_API_URL=" > .env.production

# 修改 next.config 跳过类型检查避免 SIGSEGV
RUN if [ -f next.config.ts ]; then \
    sed -i 's/const nextConfig: NextConfig = {/const nextConfig: NextConfig = {\n  typescript: { ignoreBuildErrors: true },\n  eslint: { ignoreDuringBuilds: true },/' next.config.ts; \
    fi

# 构建 Next.js 应用
RUN npm run build

# 生产运行阶段
FROM base AS runner
WORKDIR /app

ENV NODE_ENV=production
ENV NEXT_TELEMETRY_DISABLED=1

# 创建非 root 用户
RUN groupadd --system --gid 1001 nodejs
RUN useradd --system --uid 1001 nextjs

# 复制构建产物
COPY --from=builder /app/public ./public
COPY --from=builder --chown=nextjs:nodejs /app/.next/standalone ./
COPY --from=builder --chown=nextjs:nodejs /app/.next/static ./.next/static

USER nextjs

EXPOSE 3000

ENV PORT=3000
ENV HOSTNAME="0.0.0.0"

CMD ["node", "server.js"]
