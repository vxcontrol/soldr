import { Component, Input } from '@angular/core';

import { ModelsModuleS, ModelsOptionsActions, ModelsOptionsFields } from '@soldr/api';

import { LanguageService } from '../../services';

@Component({
    selector: 'soldr-action-info',
    templateUrl: './action-info.component.html',
    styleUrls: ['./action-info.component.scss']
})
export class ActionInfoComponent {
    @Input() action: ModelsOptionsActions;
    @Input() fields: ModelsOptionsFields[];
    @Input() module: ModelsModuleS;
    @Input() unavailable: boolean;

    language$ = this.languageService.current$;

    constructor(private languageService: LanguageService) {}
}
