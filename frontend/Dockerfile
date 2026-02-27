FROM node:25.6.1-alpine3.22 AS deps

WORKDIR /app
COPY package.json yarn.lock .yarnrc.yml .yarn ./

RUN npm uninstall -g yarn pnpm
RUN npm install --force -g corepack@latest
RUN corepack enable
RUN yarn install

FROM node:25.6.1-alpine3.22 AS builder

WORKDIR /app

COPY . .
COPY --from=deps /app/node_modules ./node_modules

ARG API_BASE
ENV API_BASE ${API_BASE}

RUN npm uninstall -g yarn pnpm
RUN npm install --force -g corepack@latest
RUN corepack enable
RUN yarn build

FROM nginx:1.29-alpine AS server

COPY --from=builder /app/dist /usr/share/nginx/html

EXPOSE 80

CMD ["nginx", "-g", "daemon off;"]