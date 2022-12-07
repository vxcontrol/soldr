# SOLDR WEB UI

## Development server

Create a file `proxy.conf.json` in the root directory with target and secure options, `localhost` by default.
```json
{
  "/api": {
    "target": "https://domain_name",
    "secure": false
  }
}
```
**Docs**: https://angular.io/guide/build#proxying-to-a-backend-server

Run `npm run start` for a dev server. Navigate to http://localhost:4200/. The app will automatically reload if you
change any of the source files.

## Build

Run `npm run build` to build the project. The build artifacts will be stored in the `dist/` directory.

## Running unit tests

Run `npm run test` to execute the unit tests via [Jest](https://jestjs.io).

## Understand your workspace

Run `nx dep-graph` to see a diagram of the dependencies of your projects.
