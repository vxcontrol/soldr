import { Component, Inject, Input, TemplateRef } from '@angular/core';

import { PERMISSIONS_TOKEN } from '@soldr/core';
import { Event } from '@soldr/models';

import { LanguageService } from '../../services';
import { ProxyPermission, ViewMode } from '../../types';

@Component({
    selector: 'soldr-event-info',
    templateUrl: './event-info.component.html',
    styleUrls: ['./event-info.component.scss']
})
export class EventInfoComponent {
    @Input() event: Event;
    @Input() moduleLink: TemplateRef<any>;
    @Input() viewMode: ViewMode;

    language$ = this.languageService.current$;
    viewModeEnum = ViewMode;

    constructor(
        private languageService: LanguageService,
        @Inject(PERMISSIONS_TOKEN) public permitted: ProxyPermission
    ) {}
}
