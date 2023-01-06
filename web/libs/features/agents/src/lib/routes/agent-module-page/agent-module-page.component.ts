import { ChangeDetectionStrategy, Component, Inject, OnDestroy, OnInit } from '@angular/core';
import { ActivatedRoute } from '@angular/router';
import { TranslocoService } from '@ngneat/transloco';
import { combineLatest, filter, map, startWith, Subscription, switchMap } from 'rxjs';

import { ModelsModuleA } from '@soldr/api';
import { PERMISSIONS_TOKEN } from '@soldr/core';
import { AgentModuleState, defaultAgentModuleState } from '@soldr/features/agents';
import { Agent, AgentModule } from '@soldr/models';
import {
    LanguageService,
    mergeDeep,
    PageTitleService,
    ProxyPermission,
    STATE_STORAGE_TOKEN,
    StateStorage,
    ViewMode
} from '@soldr/shared';
import { AgentCardFacade } from '@soldr/store/agents';
import { ModulesInstancesFacade } from '@soldr/store/modules-instances';
import { SharedFacade } from '@soldr/store/shared';

@Component({
    selector: 'soldr-agent-module-page',
    templateUrl: './agent-module-page.component.html',
    styleUrls: ['./agent-module-page.component.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class AgentModulePageComponent implements OnInit, OnDestroy {
    agent$ = this.agentCardFacade.agent$;
    isLoadingAgent$ = this.agentCardFacade.isLoadingAgent$;
    language$ = this.languageService.current$;
    module$ = this.agentCardFacade.agentModules$.pipe(
        filter((agentModules: AgentModule[]) => !!agentModules.length),
        map((agentModules: AgentModule[]) =>
            agentModules.find(
                (agentModule: AgentModule) => agentModule.info.name === this.activatedRoute.snapshot.params.moduleName
            )
        )
    );
    isModuleSupportOS$ = combineLatest([this.module$, this.agent$]).pipe(
        filter(([module, agent]) => !!module && !!agent),
        map(([module, agent]: [ModelsModuleA, Agent]) =>
            Object.keys(module.info.os).includes(Object.keys(agent.info.os)[0])
        ),
        startWith(true)
    );
    pageState: AgentModuleState;
    subscription = new Subscription();
    viewModeEnum = ViewMode;

    constructor(
        private activatedRoute: ActivatedRoute,
        private agentCardFacade: AgentCardFacade,
        private languageService: LanguageService,
        private modulesInstancesFacade: ModulesInstancesFacade,
        private pageTitleService: PageTitleService,
        private sharedFacade: SharedFacade,
        private transloco: TranslocoService,
        @Inject(PERMISSIONS_TOKEN) public permitted: ProxyPermission,
        @Inject(STATE_STORAGE_TOKEN) private stateStorage: StateStorage
    ) {
        const { hash } = this.activatedRoute.snapshot.params as Record<string, string>;
        this.agentCardFacade.fetchAgent(hash);
    }

    ngOnInit(): void {
        this.defineTitle();

        this.pageState = mergeDeep(
            defaultAgentModuleState(),
            (this.stateStorage.loadState('agentModule.view') as AgentModuleState) || {}
        );

        const agentSubscription = this.agent$.pipe(filter(Boolean)).subscribe((agent: Agent) => {
            const { moduleName } = this.activatedRoute.snapshot.params as Record<string, string>;

            this.modulesInstancesFacade.init(ViewMode.Agents, agent.id, moduleName);
            this.modulesInstancesFacade.fetchEvents();
        });
        this.subscription.add(agentSubscription);
    }

    ngOnDestroy(): void {
        this.subscription.unsubscribe();
    }

    private defineTitle() {
        const moduleNameTitle = combineLatest([this.module$, this.language$]).pipe(
            filter(([module]) => Boolean(module)),
            switchMap(([module, lang]) =>
                this.transloco.selectTranslate(
                    'Agents.PageTitle.Text.Module',
                    { module: module.locale.module[lang].title },
                    'agents'
                )
            )
        );
        const agentNameTitle = this.agent$.pipe(
            filter(Boolean),
            switchMap((agent) =>
                this.transloco.selectTranslate('Agents.PageTitle.Text.Agent', { agent: agent.description }, 'agents')
            )
        );

        const titlesSubscription = combineLatest([
            moduleNameTitle,
            this.transloco.selectTranslate('Agents.PageTitle.Text.Modules', {}, 'agents'),
            agentNameTitle,
            this.transloco.selectTranslate('Agents.PageTitle.Text.Agents', {}, 'agents'),
            this.sharedFacade.selectedServiceName$,
            this.transloco.selectTranslate('Shared.Pseudo.PageTitle.ApplicationName', {}, 'shared')
        ])
            .pipe(map((segments) => segments.filter(Boolean)))
            .subscribe((segments) => this.pageTitleService.setTitle(segments));

        this.subscription.add(titlesSubscription);
    }
}
