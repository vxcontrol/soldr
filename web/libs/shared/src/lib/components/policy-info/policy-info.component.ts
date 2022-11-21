import { Component, EventEmitter, Inject, Input, OnChanges, Output } from '@angular/core';
import { ThemePalette } from '@ptsecurity/mosaic/core';

import { PERMISSIONS_TOKEN } from '@soldr/core';
import { Policy, PolicyModule } from '@soldr/models';
import { LanguageService, ProxyPermission, sortGroups, sortModules, sortTags, ViewMode } from '@soldr/shared';

@Component({
    selector: 'soldr-policy-info',
    templateUrl: './policy-info.component.html',
    styleUrls: ['./policy-info.component.scss']
})
export class PolicyInfoComponent implements OnChanges {
    @Input() modules: PolicyModule[];
    @Input() policy: Policy;
    @Input() showGroups = false;
    @Input() showModules = false;
    @Input() state: any;
    @Input() stateStorageKey: string;

    @Output() selectTag = new EventEmitter<string>();

    modulesConsistencyByName: Record<string, boolean> = {};
    modulesUpgradesByName: Record<string, boolean> = {};
    language$ = this.languageService.current$;
    sortGroups = sortGroups;
    sortModules = sortModules;
    sortTags = sortTags;
    themePalette = ThemePalette;
    viewModeEnum = ViewMode;

    constructor(
        private languageService: LanguageService,
        @Inject(PERMISSIONS_TOKEN) public permitted: ProxyPermission
    ) {}

    get canShowOs() {
        return Object.keys(this.policy?.info?.os).length > 0;
    }

    ngOnChanges(): void {
        this.modulesConsistencyByName = {};
        this.modulesUpgradesByName = {};

        for (const module of this.policy?.details.modules || []) {
            const moduleName = module.info.name;
            const dependenciesByModuleName = (this.policy?.details?.dependencies || []).filter(
                // eslint-disable-next-line @typescript-eslint/naming-convention
                ({ source_module_name }) => source_module_name === moduleName
            );
            this.modulesConsistencyByName[moduleName] =
                dependenciesByModuleName.length === 0 || dependenciesByModuleName.every(({ status }) => !!status);
        }

        for (const module of this.modules || []) {
            const moduleName = module.info.name;
            this.modulesUpgradesByName[moduleName] = module.details?.update;
        }
    }
}
