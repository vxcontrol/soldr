import {
    Component,
    ContentChild,
    EventEmitter,
    Inject,
    Input,
    OnChanges,
    OnDestroy,
    OnInit,
    Output,
    SimpleChanges,
    ViewChild,
    ViewEncapsulation
} from '@angular/core';
import { TranslocoService } from '@ngneat/transloco';
import { McButton } from '@ptsecurity/mosaic/button';
import { ThemePalette } from '@ptsecurity/mosaic/core';
import { McPopoverComponent } from '@ptsecurity/mosaic/popover';
import {
    BehaviorSubject,
    combineLatest,
    combineLatestWith,
    filter,
    from,
    map,
    Observable,
    pairwise,
    shareReplay,
    Subject,
    Subscription,
    switchMap,
    toArray
} from 'rxjs';

import { ErrorResponse } from '@soldr/api';
import { PERMISSIONS_TOKEN } from '@soldr/core';
import { Agent, Group } from '@soldr/models';
import { LanguageService, ModalInfoService, ProxyPermission, sortGroups } from '@soldr/shared';

@Component({
    selector: 'soldr-move-to-group',
    templateUrl: './move-to-group.component.html',
    styleUrls: ['./move-to-group.component.scss'],
    encapsulation: ViewEncapsulation.None
})
export class MoveToGroupComponent implements OnInit, OnChanges, OnDestroy {
    @Input() agents: Agent[];
    @Input() allGroups: Group[];
    @Input() isDeletingFromGroup: boolean;
    @Input() isLoading: boolean;
    @Input() isMovingAgents: boolean;
    @Input() moveToGroupError: ErrorResponse;

    @Output() afterMove = new EventEmitter<void>();
    @Output() moveToGroups = new EventEmitter<{ ids: number[]; groupId: number }>();
    @Output() moveToNewGroups = new EventEmitter<{ ids: number[]; groupName: string }>();
    @Output() afterOpen = new EventEmitter<void>();

    @ContentChild(McButton) button: McButton;

    agentIds: number[] = [];
    foundGroups$: Observable<Group[]>;
    canCreateGroup$: Observable<boolean>;
    groupSearch = new Subject<string>();
    groupSearch$ = this.groupSearch.asObservable().pipe(shareReplay({ bufferSize: 1, refCount: true }));
    isAgentsInGroup: boolean;
    language$: Observable<string>;
    searchValue: string;
    selectedGroupId: number[];
    subscription = new Subscription();
    themePalette = ThemePalette;
    creatingGroup: string;
    currentGroupId?: number;
    sortGroups = sortGroups;

    readonly newGroupId = -1;
    private inputAllGroups$ = new BehaviorSubject<Group[]>([]);
    private inputIsDeletingFromGroup$ = new BehaviorSubject<boolean>(false);
    private inputIsMovingAgents$ = new BehaviorSubject<boolean>(false);
    private inputMoveToGroupError$ = new BehaviorSubject<ErrorResponse>(undefined);

    @ViewChild('popover') popover: McPopoverComponent;
    @ViewChild('searchInput') searchInput: HTMLInputElement;

    constructor(
        private languageServe: LanguageService,
        private modalInfoService: ModalInfoService,
        private transloco: TranslocoService,
        @Inject(PERMISSIONS_TOKEN) public permitted: ProxyPermission
    ) {}

    ngOnInit(): void {
        this.language$ = this.languageServe.current$;

        this.foundGroups$ = combineLatest([this.groupSearch$, this.inputAllGroups$, this.languageServe.current$]).pipe(
            switchMap(([search, groups, language]) =>
                from(groups).pipe(
                    filter((group) => group.info.name[language].toLowerCase().includes(search.toLowerCase())),
                    toArray()
                )
            )
        );
        this.canCreateGroup$ = combineLatest([
            this.groupSearch$,
            this.inputAllGroups$,
            this.languageServe.current$
        ]).pipe(
            map(
                ([search, groups, language]) =>
                    search?.length > 0 && groups.filter((group) => group.info.name[language] === search).length === 0
            )
        );

        this.groupSearch.next('');

        const closePopoverAndRefreshDataSubscription = this.inputIsMovingAgents$
            .pipe(pairwise(), combineLatestWith(this.inputIsDeletingFromGroup$, this.inputMoveToGroupError$))
            .subscribe(([[oldValue, newValue], isDeletingFromGroup, moveToGroupError]) => {
                if (oldValue && !newValue) {
                    this.popover.hide(0);
                    if (!moveToGroupError) {
                        this.afterMove.emit();
                    } else {
                        const errorText = isDeletingFromGroup
                            ? this.transloco.translate('agents.Agents.EditAgent.ErrorText.DeleteFromGroupAgent')
                            : this.transloco.translate('agents.Agents.EditAgent.ErrorText.MoveToGroupAgent');
                        this.modalInfoService.openErrorInfoModal(errorText);
                    }
                }
            });
        this.subscription.add(closePopoverAndRefreshDataSubscription);
    }

    ngOnChanges({ agents, allGroups, isDeletingFromGroup, isMovingAgents, moveToGroupError }: SimpleChanges): void {
        if (agents?.currentValue) {
            this.agentIds = this.agents.map(({ id }) => id);
            const groupsIds = Array.from(
                this.agents.reduce((acc, agent) => {
                    acc.add(agent.group_id);

                    return acc;
                }, new Set())
            );

            if (groupsIds.length === 1) {
                this.currentGroupId = groupsIds[0] as number;
            }

            this.isAgentsInGroup = this.agents.some((agent) => agent.group_id !== 0);
        }

        if (allGroups) {
            this.inputAllGroups$.next(allGroups.currentValue as Group[]);
        }

        if (isDeletingFromGroup) {
            this.inputIsDeletingFromGroup$.next(isDeletingFromGroup.currentValue as boolean);
        }

        if (isMovingAgents) {
            this.inputIsMovingAgents$.next(isMovingAgents.currentValue as boolean);
        }

        if (moveToGroupError) {
            this.inputMoveToGroupError$.next(moveToGroupError.currentValue as ErrorResponse);
        }
    }

    ngOnDestroy(): void {
        this.subscription.unsubscribe();
    }

    onChangeVisiblePopover(value: boolean) {
        this.groupSearch.next('');
        this.selectedGroupId = [];
        this.creatingGroup = '';

        if (value) {
            this.afterOpen.emit();
        } else {
            this.searchValue = '';
        }
    }

    deleteFromGroup() {
        this.moveToGroups.emit({ ids: this.agentIds, groupId: 0 });
        this.popover.hide(0);
    }

    moveToGroup() {
        if (this.selectedGroupId[0] === this.newGroupId) {
            this.moveToNewGroups.emit({ ids: this.agentIds, groupName: this.creatingGroup });
        } else {
            this.moveToGroups.emit({ ids: this.agentIds, groupId: this.selectedGroupId[0] });
        }
    }

    cancel() {
        this.popover.hide(0);
    }

    addGroupToList(groupName: string) {
        this.creatingGroup = groupName;
        this.searchInput.value = '';
        this.groupSearch.next('');
        this.selectedGroupId = [this.newGroupId];
    }
}
