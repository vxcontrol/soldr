import { Component, EventEmitter, Inject, Input, OnChanges, Output, SimpleChanges } from '@angular/core';
import { ThemePalette } from '@ptsecurity/mosaic/core';

import { PERMISSIONS_TOKEN } from '@soldr/core';
import { Group, GroupModule } from '@soldr/models';
import { LanguageService, ProxyPermission, ViewMode } from '@soldr/shared';
import { sortModules, sortPolicies, sortTags } from 'libs/shared/src/lib/utils/sort';

const HUNDRED_PERCENT = 100;

@Component({
    selector: 'soldr-group-info',
    templateUrl: './group-info.component.html',
    styleUrls: ['./group-info.component.scss']
})
export class GroupInfoComponent implements OnChanges {
    @Input() group: Group;
    @Input() isShortView = false;
    @Input() modules?: GroupModule[];
    @Input() state?: any;
    @Input() stateStorageKey?: string;

    @Output() selectTag = new EventEmitter<string>();

    modulesConsistencyByName: Record<string, boolean> = {};
    modulesUpgradesByName: Record<string, boolean> = {};
    language$ = this.languageService.current$;
    sortModules = sortModules;
    sortPolicies = sortPolicies;
    sortTags = sortTags;
    themePalette = ThemePalette;
    viewModeEnum = ViewMode;

    provisionerCount = 0;
    collectorCount = 0;
    detectorCount = 0;

    provisionerPercent: number;
    collectorPercent: number;
    detectorPercent: number;

    constructor(
        private languageService: LanguageService,
        @Inject(PERMISSIONS_TOKEN) public permitted: ProxyPermission
    ) {}

    ngOnChanges({ group }: SimpleChanges): void {
        this.modulesConsistencyByName = {};
        this.modulesUpgradesByName = {};

        for (const module of this.group?.details.modules || []) {
            const moduleName = module.info.name;
            const dependenciesByModuleName = (this.group?.details?.dependencies || []).filter(
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

        if (group?.currentValue) {
            const modulesCount = this.group.details?.modules?.length;
            this.provisionerCount = this.getModulesCountByTag('provisioner');
            this.provisionerPercent = this.getPercent(this.provisionerCount, modulesCount);

            this.collectorCount = this.getModulesCountByTag('collector');
            this.collectorPercent = this.getPercent(this.collectorCount, modulesCount);

            this.detectorCount = this.getModulesCountByTag('detector');
            this.detectorPercent = this.getPercent(this.detectorCount, modulesCount);
        }
    }

    getModulesCountByTag(tagName: string) {
        return this.group.details?.modules?.filter(({ info }) => info?.tags.find((tag) => tag === tagName)).length;
    }

    getPercent(dividend: number, divisor: number) {
        return Math.round((dividend / divisor) * HUNDRED_PERCENT);
    }

    getFiltrationByGroup() {
        return JSON.stringify([{ field: 'group_id', value: [this.group?.id] }]);
    }
}
