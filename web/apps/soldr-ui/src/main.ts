import { enableProdMode } from '@angular/core';
import { platformBrowserDynamic } from '@angular/platform-browser-dynamic';
import * as Handlebars from 'handlebars/dist/cjs/handlebars';

import { environment } from '@soldr/environments';

import { AppModule } from './app/app.module';

if (environment.production) {
    enableProdMode();
}

Handlebars.registerHelper('json', (context: any) => JSON.stringify(context));

platformBrowserDynamic()
    .bootstrapModule(AppModule)
    .catch((err) => console.error(err));
