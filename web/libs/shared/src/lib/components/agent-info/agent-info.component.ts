import { Component, EventEmitter, Inject, Input, OnChanges, Output } from '@angular/core';
import { ThemePalette } from '@ptsecurity/mosaic/core';

import { PERMISSIONS_TOKEN } from '@soldr/core';
import { Agent, AgentModule, AgentUpgradeTask } from '@soldr/models';
import { LanguageService, ProxyPermission, sortModules, sortTags, ViewMode } from '@soldr/shared';

@Component({
    selector: 'soldr-agent-info',
    templateUrl: './agent-info.component.html',
    styleUrls: ['./agent-info.component.scss']
})
export class AgentInfoComponent implements OnChanges {
    @Input() agent: Agent;
    @Input() latestBinaryVersion: string;
    @Input() showModules: boolean;
    @Input() modules: AgentModule[];
    @Input() state: any;
    @Input() stateStorageKey: string;
    @Input() isUpgradingAgents: boolean;
    @Input() isCancelUpgradingAgent: boolean;

    @Output() upgradeAgents = new EventEmitter<{ agents: Agent[]; version: string }>();
    @Output() cancelUpgradeAgent = new EventEmitter<{ hash: string; task: AgentUpgradeTask }>();
    @Output() selectTag = new EventEmitter<string>();
    @Output() refresh = new EventEmitter();

    modulesConsistencyByName: Record<string, boolean> = {};
    modulesUpgradesByName: Record<string, boolean> = {};
    language$ = this.languageService.current$;
    sortModules = sortModules;
    sortTags = sortTags;
    themePalette = ThemePalette;
    viewModeEnum = ViewMode;

    constructor(
        private languageService: LanguageService,
        @Inject(PERMISSIONS_TOKEN) public permitted: ProxyPermission
    ) {}

    ngOnChanges(): void {
        this.modulesConsistencyByName = {};
        this.modulesUpgradesByName = {};

        for (const module of this.agent?.details.modules || []) {
            const moduleName = module.info.name;
            const dependenciesByModuleName = (this.agent?.details?.dependencies || []).filter(
                // eslint-disable-next-line @typescript-eslint/naming-convention
                ({ module_name }) => module_name === moduleName
            );
            this.modulesConsistencyByName[moduleName] =
                dependenciesByModuleName.length === 0 || dependenciesByModuleName.every(({ status }) => !!status);
        }

        for (const module of this.modules || []) {
            const moduleName = module.info.name;
            this.modulesUpgradesByName[moduleName] = module.details?.update;
        }
    }

    onRefresh() {
        this.refresh.emit();
    }

    upgrade(event: { agents: Agent[]; version: string }) {
        this.upgradeAgents.emit(event);
    }

    cancelUpgrade(event: { hash: string; task: AgentUpgradeTask }) {
        this.cancelUpgradeAgent.emit(event);
    }
}
