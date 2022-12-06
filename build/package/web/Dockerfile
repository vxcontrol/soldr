FROM node:16 as build
ARG CYPRESS_INSTALL_BINARY=""
WORKDIR /src/app
COPY web/. .
# Install npm dependencies and run build
RUN npm install --legacy-peer-deps && npm run build --prod

FROM nginxinc/nginx-unprivileged:1.18-alpine

COPY --from=build /src/app/dist/apps/soldr-ui /opt/soldr/www
COPY --from=build /src/app/conf/nginx_base.conf /etc/nginx/conf.d/default.conf
EXPOSE 8080
CMD ["nginx", "-g", "daemon off;"]
