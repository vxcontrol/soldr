import { ChangeDetectionStrategy, Component, Inject, OnDestroy, OnInit } from '@angular/core';
import { ActivatedRoute } from '@angular/router';
import { TranslocoService } from '@ngneat/transloco';
import { combineLatest, filter, map, Subscription, switchMap } from 'rxjs';

import { PERMISSIONS_TOKEN } from '@soldr/core';
import {
    LanguageService,
    PageTitleService,
    STATE_STORAGE_TOKEN,
    StateStorage,
    ViewMode,
    mergeDeep,
    ProxyPermission
} from '@soldr/shared';
import { GroupsFacade } from '@soldr/store/groups';
import { ModulesInstancesFacade } from '@soldr/store/modules-instances';
import { SharedFacade } from '@soldr/store/shared';

import { defaultGroupModuleState, GroupModuleState } from '../../utils';

@Component({
    selector: 'soldr-group-module-page',
    templateUrl: './group-module-page.component.html',
    styleUrls: ['./group-module-page.component.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class GroupModulePageComponent implements OnInit, OnDestroy {
    group$ = this.groupsFacade.group$;
    isLoadingGroup$ = this.groupsFacade.isLoadingGroup$;
    isLoadingModule$ = this.modulesInstancesFacade.isLoadingModule$;
    moduleEventsGridColumnFilterItems$ = this.groupsFacade.moduleEventsGridColumnFilterItems$;
    language$ = this.languageService.current$;
    module$ = this.modulesInstancesFacade.module$;
    pageState: GroupModuleState;
    subscription = new Subscription();
    viewModeEnum = ViewMode;

    constructor(
        private activatedRoute: ActivatedRoute,
        private groupsFacade: GroupsFacade,
        private languageService: LanguageService,
        private modulesInstancesFacade: ModulesInstancesFacade,
        private pageTitleService: PageTitleService,
        private sharedFacade: SharedFacade,
        private transloco: TranslocoService,
        @Inject(PERMISSIONS_TOKEN) public permitted: ProxyPermission,
        @Inject(STATE_STORAGE_TOKEN) private stateStorage: StateStorage
    ) {
        const { hash } = this.activatedRoute.snapshot.params as Record<string, string>;
        this.groupsFacade.fetchGroup(hash);
    }

    ngOnInit(): void {
        this.defineTitle();

        this.pageState = mergeDeep(
            defaultGroupModuleState(),
            (this.stateStorage.loadState('groupModule.view') as GroupModuleState) || {}
        );

        const groupSubscription = this.group$.pipe(filter(Boolean)).subscribe((group) => {
            const { hash, moduleName } = this.activatedRoute.snapshot.params as Record<string, string>;

            this.modulesInstancesFacade.init(ViewMode.Groups, group.id, moduleName);
            this.modulesInstancesFacade.fetchModule(hash);
            this.modulesInstancesFacade.fetchEvents();
            this.modulesInstancesFacade.fetchModuleEventsFilterItems();
        });
        this.subscription.add(groupSubscription);

        this.sharedFacade.fetchAllAgents();
    }

    ngOnDestroy(): void {
        this.subscription.unsubscribe();
    }

    private defineTitle() {
        const moduleNameTitle = combineLatest([this.module$, this.language$]).pipe(
            filter(([module]) => Boolean(module)),
            switchMap(([module, lang]) =>
                this.transloco.selectTranslate(
                    'Groups.PageTitle.Text.Module',
                    { module: module.locale.module[lang].title },
                    'groups'
                )
            )
        );
        const groupNameTitle = combineLatest([this.group$, this.language$]).pipe(
            filter(([group]) => Boolean(group)),
            switchMap(([group, lang]) =>
                this.transloco.selectTranslate(
                    'Groups.PageTitle.Text.Group',
                    { group: group.info.name[lang] },
                    'groups'
                )
            )
        );

        const titlesSubscription = combineLatest([
            moduleNameTitle,
            this.transloco.selectTranslate('Groups.PageTitle.Text.Modules', {}, 'groups'),
            groupNameTitle,
            this.transloco.selectTranslate('Groups.PageTitle.Text.Groups', {}, 'groups'),
            this.sharedFacade.selectedServiceName$,
            this.transloco.selectTranslate('Shared.Pseudo.PageTitle.ApplicationName', {}, 'shared')
        ])
            .pipe(map((segments) => segments.filter(Boolean)))
            .subscribe((segments) => this.pageTitleService.setTitle(segments));

        this.subscription.add(titlesSubscription);
    }
}
