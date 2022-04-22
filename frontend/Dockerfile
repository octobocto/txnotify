# Inspired by https://mherman.org/blog/dockerizing-a-react-app/
FROM node:12-alpine AS builder

COPY package.json yarn.lock ./

RUN yarn --frozen-lockfiles

COPY . ./

RUN TSC_COMPILE_ON_ERROR=false yarn build

FROM nginx:alpine

WORKDIR /usr/share/nginx/html

COPY nginx.conf /etc/nginx/conf.d/default.conf

COPY --from=builder build .
COPY --from=builder ./env.sh .
COPY --from=builder ./.env.runtime .

RUN chmod +x env.sh

CMD ["/bin/sh", "-c", "/usr/share/nginx/html/env.sh > env-config.js && nginx -g \"daemon off;\""]