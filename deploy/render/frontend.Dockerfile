FROM node:24-bookworm AS deps

WORKDIR /app
RUN npm install -g yarn@1.22.22
COPY templates/frontend-next-shadcn/package.json templates/frontend-next-shadcn/yarn.lock ./
RUN yarn install --frozen-lockfile

FROM deps AS builder

ARG NEXT_PUBLIC_ADMIN_API_URL
ENV NEXT_PUBLIC_ADMIN_API_URL=$NEXT_PUBLIC_ADMIN_API_URL
COPY templates/frontend-next-shadcn .
RUN yarn run build

FROM node:24-bookworm AS runner

WORKDIR /app
ENV NODE_ENV=production
ENV PORT=3000
COPY --from=builder /app/package.json ./package.json
COPY --from=builder /app/.next ./.next
COPY --from=builder /app/public ./public
COPY --from=builder /app/node_modules ./node_modules

EXPOSE 3000
CMD ["yarn", "run", "start", "-p", "3000"]
