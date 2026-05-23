FROM node:24-bookworm AS deps
WORKDIR /app
COPY templates/frontend-next-shadcn/package.json templates/frontend-next-shadcn/package-lock.json ./
RUN npm ci

FROM deps AS builder
COPY templates/frontend-next-shadcn .
RUN npm run build

FROM node:24-bookworm AS runner
WORKDIR /app
ENV NODE_ENV=production
COPY --from=builder /app/package.json ./package.json
COPY --from=builder /app/.next ./.next
COPY --from=builder /app/public ./public
COPY --from=builder /app/node_modules ./node_modules
EXPOSE 3000
CMD ["npm", "run", "start"]
