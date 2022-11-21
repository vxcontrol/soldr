FROM node:14 as build
ARG CYPRESS_INSTALL_BINARY=""
WORKDIR /src/app
COPY . .
# Install yarn dependencies and run build
RUN yarn install && yarn build --prod

FROM nginxinc/nginx-unprivileged:1.18-alpine

COPY --from=build /src/app/dist/apps/soldr-ui /opt/soldr/www
COPY --from=build /src/app/conf/nginx_base.conf /etc/nginx/conf.d/default.conf
EXPOSE 8080
CMD ["nginx", "-g", "daemon off;"]
