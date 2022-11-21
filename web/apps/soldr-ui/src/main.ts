import { enableProdMode } from '@angular/core';
import { platformBrowserDynamic } from '@angular/platform-browser-dynamic';
import * as Handlebars from 'handlebars/dist/cjs/handlebars';

import { environment } from '@soldr/environments';

import { AppModule } from './app/app.module';

// eslint-disable-next-line import/no-unassigned-import
import './define-monaco';

if (environment.production) {
    enableProdMode();
}

Handlebars.registerHelper('json', (context: any) => JSON.stringify(context));

platformBrowserDynamic()
    .bootstrapModule(AppModule)
    .catch((err) => console.error(err));
