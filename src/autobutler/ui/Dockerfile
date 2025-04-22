FROM oven/bun:1 AS builder

WORKDIR /app

COPY package*.json ./
COPY bun.lock ./

RUN bun install

COPY . .

RUN bun run build

FROM oven/bun:1-slim AS production

WORKDIR /app

# Copy built files to a location not affected by volume mounting
COPY --from=builder /app/.output /usr/local/app/.output

EXPOSE 3000

ENV NODE_ENV=production
ENV PORT=3000

# Use the absolute path to the entry point
CMD ["bun", "/usr/local/app/.output/server/index.mjs"] 
