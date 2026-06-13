# Builder stage
FROM node:22-alpine AS builder

WORKDIR /app

# Enable pnpm
RUN corepack enable pnpm

# Install dependencies
COPY package.json pnpm-lock.yaml pnpm-workspace.yaml* ./
RUN pnpm install --frozen-lockfile

# Copy source and build
COPY tsconfig.json ./
COPY src/ ./src/
RUN pnpm run build

# Runner stage
FROM grafana/k6:latest

# Copy built test file
COPY --from=builder /app/dist /scripts

# Default entrypoint to run the main test suite
ENTRYPOINT ["k6", "run", "/scripts/main.js"]
