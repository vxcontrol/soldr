import { CommonModule } from '@angular/common';
import { LOCALE_ID, NgModule } from '@angular/core';
import {
    TRANSLOCO_CONFIG,
    TRANSLOCO_LOADER,
    TRANSLOCO_MISSING_HANDLER,
    TranslocoModule,
    TranslocoService
} from '@ngneat/transloco';
import { TranslocoMessageFormatModule } from '@ngneat/transloco-messageformat';
import { MC_DATE_LOCALE } from '@ptsecurity/cdk/datetime';
import { MC_LOCALE_ID } from '@ptsecurity/mosaic/core';

import { defaultTranslocoConfig } from './config';
import { mcLocaleTokensFactory } from './mc-locale-token-factory';
import { MissingKeyHandler } from './missing-key-handler';
import { TranslocoHttpLoaderService } from './transloco-http-loader.service';

@NgModule({
    exports: [TranslocoModule],
    imports: [CommonModule, TranslocoModule, TranslocoMessageFormatModule.forRoot()],
    providers: [
        { provide: TRANSLOCO_CONFIG, useValue: defaultTranslocoConfig },
        { provide: TRANSLOCO_LOADER, useClass: TranslocoHttpLoaderService },
        { provide: TRANSLOCO_MISSING_HANDLER, useClass: MissingKeyHandler },
        {
            provide: LOCALE_ID,
            useFactory: (translocoService: TranslocoService) => translocoService.getActiveLang(),
            deps: [TranslocoService]
        },
        { provide: MC_LOCALE_ID, useFactory: mcLocaleTokensFactory }, // used for mc formatters
        { provide: MC_DATE_LOCALE, useFactory: mcLocaleTokensFactory } // used for date adapter
    ]
})
export class I18nModule {}
