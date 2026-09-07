FROM --platform=linux/amd64 node:23.3.0-alpine3.19 AS build

WORKDIR /usr/src/app

COPY . .

RUN npm ci
RUN npm run build

FROM --platform=linux/amd64 nginx:1.27-alpine3.20 AS prod

COPY --from=build /usr/src/app/static /usr/share/nginx/html/static

COPY --from=build /usr/src/app/dist /usr/share/nginx/html

# The stock default.conf serves nginx's welcome page from the wrong root and
# has no SPA fallback; replace it rather than leaving the image with no server
# block, which is what made this image unusable without an external mount.
RUN rm /etc/nginx/conf.d/default.conf
COPY --from=build /usr/src/app/nginx.conf /etc/nginx/conf.d/nginx.conf

EXPOSE 80

CMD ["nginx", "-g", "daemon off;"]