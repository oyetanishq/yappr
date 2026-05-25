# ── Stage 1: Build ──────────────────────────────────────────────────────────
FROM node:22-alpine AS builder

WORKDIR /app

# Enable pnpm
RUN corepack enable pnpm

# Install dependencies
COPY package.json pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile

# Copy source and build
COPY . .
ARG VITE_API_URL
ENV VITE_API_URL=${VITE_API_URL}
RUN pnpm run build

# ── Stage 2: Serve ──────────────────────────────────────────────────────────
FROM nginx:alpine

# Configure nginx for client-side routing (React Router)
RUN sed -i 's/try_files $uri $uri\/ =404;/try_files $uri $uri\/ \/index.html;/' /etc/nginx/conf.d/default.conf

# Copy built assets
COPY --from=builder /app/dist /usr/share/nginx/html

EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]
