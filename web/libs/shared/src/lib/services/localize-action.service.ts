import { Injectable } from '@angular/core';
import { TranslocoService } from '@ngneat/transloco';

import { ModelsEventConfigAction, ModelsOptionsActions } from '@soldr/api';
import { LanguageService, LOG_TO_DB } from '@soldr/shared';

@Injectable({ providedIn: 'root' })
export class LocalizeActionService {
    constructor(private languageService: LanguageService, private transloco: TranslocoService) {}

    localizeAction(action: ModelsEventConfigAction, globalActions: ModelsOptionsActions[]) {
        const globalAction = globalActions.find(({ name }) => action.name === name);
        const localization =
            action.name === LOG_TO_DB
                ? {
                      localizedName: this.transloco.translate('shared.Shared.ModuleConfig.Text.LogToDbAction')
                  }
                : {
                      localizedName: globalAction?.locale[this.languageService.lang].title || action.name,
                      localizedDescription: globalAction?.locale[this.languageService.lang].description || ''
                  };

        return {
            ...action,
            ...localization
        };
    }
}
