import { Component, Inject, Input } from '@angular/core';

import { ModelsGroup } from '@soldr/api';
import { LanguageService, ProxyPermission } from '@soldr/shared';
import { PERMISSIONS_TOKEN } from '@soldr/core';

@Component({
    selector: 'soldr-group-link',
    templateUrl: './group-link.component.html',
    styleUrls: ['./group-link.component.scss']
})
export class GroupLinkComponent {
    @Input() group!: ModelsGroup;

    language$ = this.language.current$;

    constructor(private language: LanguageService, @Inject(PERMISSIONS_TOKEN) public permitted: ProxyPermission) {}
}
