import {
    Component,
    ContentChild,
    Inject,
    Input,
    OnDestroy,
    OnInit,
    TemplateRef,
    ViewChild,
    ViewEncapsulation
} from '@angular/core';
import { NavigationStart, Router } from '@angular/router';
import { TranslocoService } from '@ngneat/transloco';
import { McButton } from '@ptsecurity/mosaic/button';
import { ThemePalette } from '@ptsecurity/mosaic/core';
import { McModalRef, McModalService, ModalSize } from '@ptsecurity/mosaic/modal';
import { McSidepanelPosition, McSidepanelRef, McSidepanelService } from '@ptsecurity/mosaic/sidepanel';
import {
    Observable,
    BehaviorSubject,
    Subject,
    map,
    combineLatest,
    Subscription,
    pairwise,
    first,
    combineLatestWith
} from 'rxjs';

import { ModelsModuleAShort } from '@soldr/api';
import { PERMISSIONS_TOKEN } from '@soldr/core';
import { Group, Policy } from '@soldr/models';
import { LanguageService, ModalInfoService, POLICY_LINKING_FACADE, ProxyPermission } from '@soldr/shared';
import { GroupsFacade } from '@soldr/store/groups';
import { PoliciesFacade } from '@soldr/store/policies';

export { POLICY_LINKING_FACADE } from './link-group-to-policy.tokens';

export type ListEntity = Policy | Group;
export type Conflict = { module: ModelsModuleAShort; conflictedPolicy: Policy };
export type ConflictByPolicy = { conflictedPolicy: Policy; modules: ModelsModuleAShort[] };
export type ConflictsByEntity = { [entityId: number]: Conflict[] };
export type Grouping = 'module' | 'policy';

export enum SortDirection {
    ASC = 'asc',
    DESC = 'desc'
}

export interface LinkPolicyToGroupFacade<A = ListEntity, B = ListEntity> {
    available$: Observable<A[]>;
    baseEntity$: Subject<B>;
    conflictGroup$: Observable<Group>;
    conflictPolicy$: Observable<Policy>;
    conflictedItem: Subject<A>;
    conflictsByEntityId$: Observable<ConflictsByEntity>;
    disabled$: Observable<boolean>;
    fetchData: () => void;
    groupedConflictsByModule$: Observable<Conflict[]>;
    groupedConflictsByPolicy$: Observable<ConflictByPolicy[]>;
    isLoading$: Observable<boolean>;
    items$: Observable<A[]>;
    link: (hash: string, item: A) => void;
    linked$: Observable<A[]>;
    unavailable$: Observable<A[]>;
    unlink: (hash: string, item: A) => void;
}

@Component({
    selector: 'soldr-link-group-to-policy',
    templateUrl: './link-group-to-policy.component.html',
    styleUrls: ['./link-group-to-policy.component.scss'],
    encapsulation: ViewEncapsulation.None
})
export class LinkGroupToPolicyComponent implements OnInit, OnDestroy {
    @Input() title: string;
    @Input() placeholder: string;
    @Input() linkedLabel: string;
    @Input() conflictTitle: TemplateRef<any>;

    @ViewChild('panel') template: TemplateRef<any>;
    @ViewChild('conflictsContent') conflictsContentRef: TemplateRef<any>;
    @ViewChild('conflictsFooter') conflictsFooterRef: TemplateRef<any>;
    @ContentChild(McButton) button: McButton;

    themePalette = ThemePalette;
    linked$ = this.facade.linked$;
    available$ = this.facade.available$;
    unavailable$ = this.facade.unavailable$;
    baseEntity$ = this.facade.baseEntity$;
    conflictsByEntityId$ = this.facade.conflictsByEntityId$;
    conflictGroup$ = this.facade.conflictGroup$;
    conflictPolicy$ = this.facade.conflictPolicy$;
    language$ = this.languageService.current$;
    conflictsModal: McModalRef;
    sidePanel: McSidepanelRef;
    isLoading$: Observable<boolean>;
    search$ = new BehaviorSubject('');
    grouping: Grouping = 'module';
    searchValue = '';
    sortDirectionByModule = new BehaviorSubject<SortDirection | undefined>(undefined);
    sortDirectionByModule$ = this.sortDirectionByModule.asObservable();
    sortDirectionByPolicy = new BehaviorSubject<SortDirection | undefined>(undefined);
    sortDirectionByPolicy$ = this.sortDirectionByPolicy.asObservable();
    sorting = SortDirection;
    subscription = new Subscription();
    groupedConflictsByModule$ = combineLatest([
        this.sortDirectionByModule$,
        this.facade.groupedConflictsByModule$
    ]).pipe(map(([direction, conflicts]) => conflicts?.sort(this.sortComparatorByModule(direction))));
    groupedConflictsByPolicy$ = combineLatest([
        this.sortDirectionByModule$,
        this.facade.groupedConflictsByPolicy$
    ]).pipe(map(([direction, conflicts]) => conflicts?.sort(this.sortComparatorByPolicy(direction))));
    searchPolicy = (policy: Policy, search: string): boolean =>
        policy.info.name[this.languageService.lang].toLocaleLowerCase().includes(search.toLocaleLowerCase());

    sortItems = (language: string) => (a: ListEntity, b: ListEntity) =>
        a.info.name[language].localeCompare(b.info.name[language], 'en');

    constructor(
        private groupsFacade: GroupsFacade,
        private languageService: LanguageService,
        private modalInfoService: ModalInfoService,
        private modalService: McModalService,
        private policiesFacade: PoliciesFacade,
        private router: Router,
        private sidePanelService: McSidepanelService,
        private transloco: TranslocoService,
        @Inject(POLICY_LINKING_FACADE) private facade: LinkPolicyToGroupFacade<Policy, Group>,
        @Inject(PERMISSIONS_TOKEN) public permitted: ProxyPermission
    ) {}

    ngOnDestroy(): void {
        this.subscription.unsubscribe();
    }

    ngOnInit(): void {
        this.isLoading$ = this.facade.isLoading$;

        const subscription = this.router.events.subscribe((event) => {
            if (event instanceof NavigationStart) {
                this.conflictsModal?.close();
                this.sidePanel?.close();
            }
        });
        this.subscription.add(subscription);

        const isLinkingPolicySubscription = this.policiesFacade.isLinkingPolicy$
            .pipe(pairwise(), combineLatestWith(this.policiesFacade.linkPolicyFromGroupError$))
            .subscribe(([[previous, current], linkToGroupError]) => {
                if (previous && !current && linkToGroupError) {
                    this.processedError(
                        this.transloco.translate('shared.Shared.LinkGroupToPolicy.ErrorText.LinkToGroup')
                    );
                }
            });
        this.subscription.add(isLinkingPolicySubscription);

        const isUnlinkingPolicySubscription = this.policiesFacade.isUnlinkingPolicy$
            .pipe(pairwise(), combineLatestWith(this.policiesFacade.unlinkPolicyFromGroupError$))
            .subscribe(([[previous, current], unlinkToGroupError]) => {
                if (previous && !current && unlinkToGroupError) {
                    this.processedError(
                        this.transloco.translate('shared.Shared.LinkGroupToPolicy.ErrorText.UnlinkToGroup')
                    );
                }
            });
        this.subscription.add(isUnlinkingPolicySubscription);

        const isLinkingGroupSubscription = this.groupsFacade.isLinkingGroup$
            .pipe(pairwise(), combineLatestWith(this.groupsFacade.linkGroupToPolicyError$))
            .subscribe(([[previous, current], linkToPolicyError]) => {
                if (previous && !current && linkToPolicyError) {
                    this.processedError(
                        this.transloco.translate('shared.Shared.LinkGroupToPolicy.ErrorText.LinkToPolicy')
                    );
                }
            });
        this.subscription.add(isLinkingGroupSubscription);

        const isUnlinkingGroupSubscription = this.groupsFacade.isUnlinkingGroup$
            .pipe(pairwise(), combineLatestWith(this.groupsFacade.unlinkGroupFromPolicyError$))
            .subscribe(([[previous, current], unlinkToPolicyError]) => {
                if (previous && !current && unlinkToPolicyError) {
                    this.processedError(
                        this.transloco.translate('shared.Shared.LinkGroupToPolicy.ErrorText.UnlinkToPolicy')
                    );
                }
            });
        this.subscription.add(isUnlinkingGroupSubscription);
    }

    open() {
        this.facade.fetchData();

        if (this.button.disabled) {
            return;
        }

        this.sidePanel = this.sidePanelService.open(this.template, {
            position: McSidepanelPosition.Right,
            hasBackdrop: true,
            overlayPanelClass: 'link-group-to-policy__overlay'
        });

        this.sidePanel.afterClosed().subscribe(() => {
            this.unloadData();
        });
    }

    processedError(text: string) {
        this.sidePanel.close();
        this.sidePanel
            .afterClosed()
            .pipe(first())
            .subscribe(() => this.modalInfoService.openErrorInfoModal(text));
    }

    unlink(group: Group, policy: Policy) {
        this.facade.unlink(group.hash, policy);
    }

    link(group: Group, policy: Policy) {
        this.facade.link(group.hash, policy);
    }

    private unloadData() {
        this.search$.next('');
    }

    openConflictsPopover(policy: Policy) {
        this.facade.conflictedItem.next(policy);

        this.conflictsModal = this.modalService.create({
            mcTitle: this.conflictTitle,
            mcContent: this.conflictsContentRef,
            mcSize: ModalSize.Normal,
            mcClassName: 'link-group-to-policy__modal'
        });
    }

    toggleSortByModule(currentSorting: SortDirection) {
        this.sortDirectionByModule.next(this.getNextSortDirection(currentSorting));
    }

    toggleSortByPolicy(currentSorting: SortDirection) {
        this.sortDirectionByPolicy.next(this.getNextSortDirection(currentSorting));
    }

    private getNextSortDirection(currentSorting: SortDirection) {
        return currentSorting === SortDirection.ASC
            ? SortDirection.DESC
            : currentSorting === SortDirection.DESC
            ? undefined
            : SortDirection.ASC;
    }

    private sortComparatorByModule(direction: SortDirection) {
        return (a: Conflict, b: Conflict) =>
            [SortDirection.ASC, undefined].includes(direction)
                ? a.module.locale.module[this.languageService.lang].title.localeCompare(
                      b.module.locale.module[this.languageService.lang].title
                  )
                : b.module.locale.module[this.languageService.lang].title.localeCompare(
                      a.module.locale.module[this.languageService.lang].title
                  );
    }

    private sortComparatorByPolicy(direction: SortDirection) {
        return (a: ConflictByPolicy, b: ConflictByPolicy) =>
            [SortDirection.ASC, undefined].includes(direction)
                ? a.conflictedPolicy.info.name[this.languageService.lang].localeCompare(
                      b.conflictedPolicy.info.name[this.languageService.lang]
                  )
                : b.conflictedPolicy.info.name[this.languageService.lang].localeCompare(
                      a.conflictedPolicy.info.name[this.languageService.lang]
                  );
    }
}
