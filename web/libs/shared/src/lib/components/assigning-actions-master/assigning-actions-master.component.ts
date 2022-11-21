import { Component, EventEmitter, Input, OnDestroy, OnInit, Output, TemplateRef, ViewChild } from '@angular/core';
import { TranslocoService } from '@ngneat/transloco';
import { PopUpPlacements, ThemePalette } from '@ptsecurity/mosaic/core';
import { McListSelection } from '@ptsecurity/mosaic/list';
import { McModalRef, McModalService, ModalSize } from '@ptsecurity/mosaic/modal';
import { defaultCompareValues, FlatTreeControl, McTreeFlatDataSource, McTreeFlattener } from '@ptsecurity/mosaic/tree';
import {
    BehaviorSubject,
    combineLatest,
    concatMap,
    filter,
    from,
    map,
    Observable,
    Subject,
    Subscription,
    take,
    toArray
} from 'rxjs';

import {
    DependencyType,
    ModelsDependencyItem,
    ModelsEventConfigAction,
    ModelsModuleS,
    ModelsOptionsActions
} from '@soldr/api';
import { LANGUAGES } from '@soldr/i18n';
import { SharedFacade } from '@soldr/store/shared';

import { LanguageService } from '../../services';
import { EntityModule, LOG_TO_DB, LOG_TO_DB_ACTION } from '../../types';
import { clone, difference } from '../../utils';

const LAST_STEP = 2;

enum ActionNodeType {
    Module = 'module',
    Action = 'action'
}

enum ActionFilterKind {
    All = 'all',
    AvailableForAssigning = 'available-for-assigning'
}

enum ActionGroupingKind {
    ByModule = 'by-module',
    No = 'no'
}

class ActionTreeNode {
    type: ActionNodeType;
    level: number;
    data: { action?: ModelsOptionsActions; module: ModelsModuleS };
    children?: ActionTreeNode[];
    unavailable?: boolean;
}

interface EventActionItem extends ModelsEventConfigAction {
    action: ModelsOptionsActions;
    module: ModelsModuleS;
}

@Component({
    selector: 'soldr-assigning-actions-master',
    templateUrl: './assigning-actions-master.component.html',
    styleUrls: ['./assigning-actions-master.component.scss']
})
export class AssigningActionsMasterComponent implements OnInit, OnDestroy {
    @Input() module: EntityModule;

    @Output() saveModule = new EventEmitter<EntityModule>();

    @ViewChild('body') bodyTemplate: TemplateRef<any>;
    @ViewChild('footer') footerTemplate: TemplateRef<any>;
    @ViewChild('test') test: McListSelection;

    actionsDataSource: McTreeFlatDataSource<ActionTreeNode, ActionTreeNode>;
    actionsFilter$ = new BehaviorSubject(ActionFilterKind.All);
    actionsFilterKind = ActionFilterKind;
    actionsGrouping$ = new BehaviorSubject(ActionGroupingKind.ByModule);
    actionGroupingKind = ActionGroupingKind;
    actionsTreeControl: FlatTreeControl<ActionTreeNode>;
    actionsTreeFlattener: McTreeFlattener<ActionTreeNode, ActionTreeNode>;
    availableEvents$ = new BehaviorSubject<string[]>([]);
    unavailableEvents$ = new BehaviorSubject<string[]>([]);
    assignedEvents$ = new BehaviorSubject<string[]>([]);
    isLoading$ = combineLatest([this.sharedFacade.isLoadingAllModules$, this.sharedFacade.isLoadingActions$]).pipe(
        map((values) => values.every((v) => v))
    );
    language$ = this.languageService.current$;
    modal: McModalRef;
    popUpPlacements = PopUpPlacements;
    processedAvailableEvents$: Observable<string[]>;
    processedUnavailableEvents$: Observable<string[]>;
    processedSelectedEvents$: Observable<string[]>;
    searchActions$ = new BehaviorSubject('');
    searchActions: string;
    selectedActionTreeNode: ActionTreeNode;
    step = 1;
    subscription = new Subscription();
    themePalette = ThemePalette;
    searchEvents$ = new BehaviorSubject('');
    searchEvents: string;
    searchSelectedEvents$ = new BehaviorSubject('');
    searchSelectedEvents: string;
    eventsNodeFromNotSelected: string[] = [];
    eventsNodeFromSelected: string[] = [];
    lastSelectedEvent$: Subject<string> = new Subject<string>();
    lastSelectedEventActions$: Observable<EventActionItem[]>;
    logToDbActionName = LOG_TO_DB;

    constructor(
        private modalService: McModalService,
        private languageService: LanguageService,
        private sharedFacade: SharedFacade,
        private transloco: TranslocoService
    ) {}

    ngOnInit(): void {
        this.defineActionsObservables();
        this.defineEventsObservables();
    }

    ngOnDestroy(): void {
        this.subscription.unsubscribe();
    }

    open() {
        this.actionsGrouping$.next(ActionGroupingKind.ByModule);

        this.setActionsData();
        this.setEventsData();

        this.sharedFacade.fetchAllModules();
        this.sharedFacade.fetchActions();
        this.sharedFacade.fetchFields();

        this.modal = this.modalService.create({
            mcSize: ModalSize.Large,
            mcContent: this.bodyTemplate,
            mcFooter: this.footerTemplate,
            mcBodyStyle: {
                padding: 0,
                overflow: 'hidden'
            }
        });

        this.modal.afterClose.pipe(take(1)).subscribe(() => this.close());
    }

    cancel() {
        this.modal.close();
    }

    close() {
        this.step = 1;
        this.searchActions = '';
        this.availableEvents$.next([]);
        this.unavailableEvents$.next([]);
        this.assignedEvents$.next([]);
        this.searchEvents$.next('');
        this.searchSelectedEvents$.next('');
        this.lastSelectedEvent$.next(undefined);
        this.eventsNodeFromNotSelected = [];
    }

    save() {
        this.assignedEvents$.pipe(take(1)).subscribe((events) => {
            const module = clone(this.module) as EntityModule;
            const actionName = this.selectedActionTreeNode.data.action?.name;
            const actionModuleName = this.selectedActionTreeNode.data.action?.module_name;

            for (const eventName of Object.keys(module.current_event_config)) {
                module.current_event_config[eventName].actions = module.current_event_config[eventName].actions.filter(
                    ({ name }) => name !== actionName
                );
            }

            for (const event of events) {
                if (module.current_event_config[event].actions.find(({ name }) => name === actionName)) {
                    continue;
                }

                module.current_event_config[event].actions.push({
                    name: actionName,
                    module_name: this.selectedActionTreeNode.data.action?.module_name,
                    priority: this.selectedActionTreeNode.data.action?.config.priority,
                    fields: this.selectedActionTreeNode.data.action?.config.fields
                } as ModelsEventConfigAction);
            }

            if (!module.dynamic_dependencies.find((dependency) => dependency.module_name === actionModuleName)) {
                module.dynamic_dependencies.push({
                    module_name: actionModuleName,
                    type: DependencyType.ToMakeAction
                });
            }

            this.clearDynamicDependencies(module);
            this.saveModule.emit(module);
            this.modal.close();
        });
    }

    next() {
        if (this.step < LAST_STEP) {
            this.step++;
            this.setSelectedEvents();
        }
    }

    back() {
        if (this.step > 1) {
            this.step--;
        }
    }

    hasChildInActionTree(index: number, node: ActionTreeNode) {
        return node.type === ActionNodeType.Module;
    }

    selectEvents(eventName: string) {
        const moving = Array.from(new Set([...this.eventsNodeFromNotSelected, eventName]));

        this.assignedEvents$.pipe(take(1)).subscribe((events) => {
            this.assignedEvents$.next([...events, ...moving]);
        });

        this.availableEvents$.pipe(take(1)).subscribe((events) => {
            this.availableEvents$.next(events.filter((name) => !moving.includes(name)));
            this.eventsNodeFromNotSelected = [];
        });
    }

    unselectEvent(eventName: string) {
        const moving = Array.from(new Set([...this.eventsNodeFromSelected, eventName]));

        this.assignedEvents$.pipe(take(1)).subscribe((selected) => {
            this.assignedEvents$.next(selected.filter((name) => !moving.includes(name)));
        });

        this.availableEvents$.pipe(take(1)).subscribe((events) => {
            this.availableEvents$.next([...events, ...moving]);
            this.eventsNodeFromSelected = [];
        });
    }

    onSelectAction() {
        this.setEventsData();
    }

    onSelectEventsFromNotSelected($event: any) {
        this.lastSelectedEvent$.next($event.option.value as string);
    }

    onSelectEventsFromSelected($event: any) {
        this.lastSelectedEvent$.next($event.option.value as string);
    }

    assignToAvailableEvents() {
        this.availableEvents$.pipe(take(1)).subscribe((events) => {
            const module = clone(this.module) as EntityModule;
            const actionName = this.selectedActionTreeNode.data.action?.name;
            const actionModuleName = this.selectedActionTreeNode.data.action?.module_name;

            for (const event of events) {
                if (module.current_event_config[event].actions.find(({ name }) => name === actionName)) {
                    continue;
                }

                module.current_event_config[event].actions.push({
                    name: actionName,
                    module_name: this.selectedActionTreeNode.data.action?.module_name,
                    priority: this.selectedActionTreeNode.data.action?.config.priority,
                    fields: this.selectedActionTreeNode.data.action?.config.fields
                } as ModelsEventConfigAction);
            }

            if (
                !module.dynamic_dependencies.find(
                    (dependency) =>
                        dependency.module_name === actionModuleName && dependency.type === DependencyType.ToMakeAction
                )
            ) {
                module.dynamic_dependencies.push({
                    module_name: actionModuleName,
                    type: DependencyType.ToMakeAction
                });
            }

            this.clearDynamicDependencies(module);
            this.saveModule.emit(module);
            this.modal.close();
        });
    }

    removeFromAllEvents() {
        const module = clone(this.module) as EntityModule;
        const actionName = this.selectedActionTreeNode.data.action?.name;

        for (const eventName of Object.keys(module.current_event_config)) {
            module.current_event_config[eventName].actions = module.current_event_config[eventName].actions.filter(
                ({ name }) => name !== actionName
            );
        }

        this.clearDynamicDependencies(module);
        this.saveModule.emit(module);
        this.modal.close();
    }

    get hasEventsWithSelectedAction() {
        const actionName = this.selectedActionTreeNode.data.action?.name;

        return (
            actionName &&
            Object.keys(this.module.current_event_config).some(
                (eventName) =>
                    !!this.module.current_event_config[eventName].actions.find(({ name }) => name === actionName)
            )
        );
    }

    private getLevel(node: ActionTreeNode) {
        return node.level;
    }

    private getIsExpandable(node: ActionTreeNode) {
        return node.type === ActionNodeType.Module;
    }

    private sortActionTreeNodes(lang: string) {
        return (a: ActionTreeNode, b: ActionTreeNode) =>
            a.type === ActionNodeType.Module
                ? a.data.module.locale.module[lang].title.localeCompare(b.data.module.locale.module[lang].title, 'en')
                : a.data.action?.locale[lang].title.localeCompare(b.data.action?.locale[lang].title, 'en');
    }

    private sortEventsListNodes(module: EntityModule, lang: string) {
        return (a: string, b: string) =>
            module.locale.events[a][lang].title.localeCompare(module.locale.events[b][lang].title, 'en');
    }

    private getActionAvailability(action: ModelsOptionsActions) {
        return !Object.keys(this.module?.default_event_config).some(
            (eventName) =>
                difference(this.module.default_event_config[eventName].fields, action?.config.fields).length === 0
        );
    }

    private getEventAvailability(eventName: string, action: ModelsOptionsActions) {
        return difference(this.module.default_event_config[eventName].fields, action?.config.fields).length === 0;
    }

    private setActionsData() {
        const actionsWithModulesTree$ = combineLatest([
            this.sharedFacade.allModules$,
            this.sharedFacade.optionsActions$,
            this.actionsFilter$
        ]).pipe(
            take(1),
            map(([modules, actions, filter]) =>
                modules
                    .filter(
                        (module) =>
                            Object.keys(module.default_action_config).length > 0 &&
                            this.module &&
                            this.module.info.name !== module.info.name
                    )
                    .map((module) => ({
                        type: ActionNodeType.Module,
                        level: 0,
                        data: { module },
                        children: Object.keys(module.default_action_config)
                            .map((actionName) => {
                                const action: ModelsOptionsActions = actions.find(({ name }) => name === actionName);

                                return {
                                    type: ActionNodeType.Action,
                                    level: 1,
                                    data: { action, module },
                                    unavailable: this.getActionAvailability(action)
                                };
                            })
                            .filter((node: ActionTreeNode) => filter === ActionFilterKind.All || !node.unavailable)
                            .sort(this.sortActionTreeNodes(this.languageService.lang))
                    }))
                    .filter((node: ActionTreeNode) => filter === ActionFilterKind.All || node.children?.length > 0)
                    .sort(this.sortActionTreeNodes(this.languageService.lang))
            ),
            map((modules) => {
                const action: ModelsOptionsActions = {
                    config: {
                        priority: LOG_TO_DB_ACTION.priority
                    },
                    locale: {
                        [LANGUAGES.ru]: {
                            title: this.transloco.translate('shared.Shared.ModuleConfig.ListItemText.LogToDb'),
                            description: this.transloco.translate('shared.Shared.ModuleConfig.ListItemText.LogToDb')
                        },
                        [LANGUAGES.en]: {
                            title: this.transloco.translate('shared.Shared.ModuleConfig.ListItemText.LogToDb'),
                            description: this.transloco.translate('shared.Shared.ModuleConfig.ListItemText.LogToDb')
                        }
                    },
                    module_name: 'System',
                    module_os: {},
                    name: 'system'
                };
                const module = {
                    ...this.module,
                    locale: {
                        ...this.module.locale,
                        module: {
                            [LANGUAGES.ru]: {
                                title: this.transloco.translate('shared.Shared.ModuleConfig.ListItemText.Logging'),
                                description: this.transloco.translate('shared.Shared.ModuleConfig.ListItemText.Logging')
                            },
                            [LANGUAGES.en]: {
                                title: this.transloco.translate('shared.Shared.ModuleConfig.ListItemText.Logging'),
                                description: this.transloco.translate('shared.Shared.ModuleConfig.ListItemText.Logging')
                            }
                        }
                    }
                };

                return [
                    {
                        type: ActionNodeType.Module,
                        level: 0,
                        data: { module },
                        children: [
                            {
                                type: ActionNodeType.Action,
                                level: 1,
                                data: { action, module },
                                unavailable: this.getActionAvailability(action)
                            }
                        ]
                    },
                    ...modules
                ];
            })
        );

        const onlyActionsTree$ = actionsWithModulesTree$.pipe(
            concatMap((tree) => from(tree)),
            concatMap((module) => from(module.children)),
            map((action) => ({ ...action, level: 0 })),
            toArray(),
            map((actions) => actions.sort(this.sortActionTreeNodes(this.languageService.lang)))
        );

        combineLatest([this.actionsGrouping$, actionsWithModulesTree$, onlyActionsTree$]).subscribe(
            ([grouping, treeWithModulesTree, onlyActionsTree]) => {
                this.actionsDataSource.data =
                    grouping === ActionGroupingKind.ByModule ? treeWithModulesTree : onlyActionsTree;
                this.actionsTreeControl.expandAll();
                this.selectedActionTreeNode =
                    grouping === ActionGroupingKind.ByModule
                        ? this.actionsTreeControl.dataNodes[0]?.children[0]
                        : this.actionsTreeControl.dataNodes[0];
            }
        );
    }

    private setEventsData() {
        const allEvents$ = from(Object.keys(this.module.current_event_config));

        allEvents$
            .pipe(
                filter((eventName) => this.getEventAvailability(eventName, this.selectedActionTreeNode.data.action)),
                toArray(),
                map((eventNames) => eventNames.sort(this.sortEventsListNodes(this.module, this.languageService.lang)))
            )
            .subscribe((eventNames) => this.availableEvents$.next(eventNames));

        allEvents$
            .pipe(
                filter((eventName) => !this.getEventAvailability(eventName, this.selectedActionTreeNode.data.action)),
                toArray(),
                map((eventNames) => eventNames.sort(this.sortEventsListNodes(this.module, this.languageService.lang)))
            )
            .subscribe((eventNames) => this.unavailableEvents$.next(eventNames));

        this.assignedEvents$.next([]);
    }

    private setSelectedEvents() {
        const allEvents$ = from(Object.keys(this.module.current_event_config));

        allEvents$
            .pipe(
                filter(
                    (eventName) =>
                        !!this.module.current_event_config[eventName].actions.find(
                            ({ name }) => name === this.selectedActionTreeNode.data.action?.name
                        )
                )
            )
            .subscribe((eventName) => {
                this.selectEvents(eventName);
            });

        combineLatest([this.processedAvailableEvents$, this.processedUnavailableEvents$])
            .pipe(
                filter(([v1, v2]) => v1.length > 0 || v2.length > 0),
                take(1)
            )
            .subscribe(([v1, v2]) => {
                this.lastSelectedEvent$.next(v1[0] || v2[0]);
                this.eventsNodeFromNotSelected.push(v1[0] || '');
            });
    }

    private defineActionsObservables() {
        this.actionsTreeFlattener = new McTreeFlattener(
            (node: ActionTreeNode) => node,
            this.getLevel,
            this.getIsExpandable,
            (node: ActionTreeNode) => node.children
        );

        this.actionsTreeControl = new FlatTreeControl<ActionTreeNode>(
            this.getLevel,
            this.getIsExpandable,
            (node: ActionTreeNode) => node,
            (node: ActionTreeNode) =>
                node.type === ActionNodeType.Module
                    ? node.data.module.locale.module[this.languageService.lang].title
                    : node.data.action.locale[this.languageService.lang].title,
            defaultCompareValues,
            // eslint-disable-next-line @typescript-eslint/no-unsafe-argument
            this.compareViewValues.bind(this),
            (node: ActionTreeNode) => node.type === ActionNodeType.Module
        );

        this.actionsDataSource = new McTreeFlatDataSource(this.actionsTreeControl, this.actionsTreeFlattener);
        this.actionsDataSource.data = [];

        const groupingSubscription = this.actionsGrouping$.subscribe(() => {
            this.searchActions$.next('');
        });
        this.subscription.add(groupingSubscription);

        const searchActionsSubscription = this.searchActions$.subscribe((value) => {
            this.actionsTreeControl?.filterNodes(value);
            this.searchActions = value;
        });
        this.subscription.add(searchActionsSubscription);

        const filtrationSubscription = this.actionsFilter$.subscribe(() => {
            this.searchActions$.next('');
            this.setActionsData();
        });
        this.subscription.add(filtrationSubscription);
    }

    private compareViewValues(nodeName: string, search: string): boolean {
        const data = this.actionsTreeControl.dataNodes.find(
            (node) => this.actionsTreeControl.getViewValue(node) === nodeName
        );
        const ids = data.type === ActionNodeType.Module ? data.data[data.type].info.name : data.data[data.type].name;

        if (!search) {
            return true;
        }

        return nodeName.toLowerCase().includes(search.toLowerCase()) || ids.includes(search.toLowerCase());
    }

    private defineEventsObservables() {
        const processEvent: (v: [string, string[]]) => string[] = ([search, events]) =>
            events
                .filter(
                    (eventName: string) =>
                        !search ||
                        eventName.toLowerCase().includes(search.toLowerCase()) ||
                        this.module.locale.events[eventName][this.languageService.lang].title
                            .toLowerCase()
                            .includes(search.toLowerCase())
                )
                .sort(this.sortEventsListNodes(this.module, this.languageService.lang));

        this.processedAvailableEvents$ = combineLatest([this.searchEvents$, this.availableEvents$]).pipe(
            map(processEvent)
        );

        this.processedUnavailableEvents$ = combineLatest([this.searchEvents$, this.unavailableEvents$]).pipe(
            map(processEvent)
        );

        this.processedSelectedEvents$ = combineLatest([this.searchSelectedEvents$, this.assignedEvents$]).pipe(
            map(processEvent)
        );

        this.lastSelectedEventActions$ = combineLatest([
            this.lastSelectedEvent$,
            this.sharedFacade.optionsActions$,
            this.sharedFacade.allModules$
        ]).pipe(
            map(([lastSelectedEvent, actions, modules]) =>
                this.module.current_event_config[lastSelectedEvent]?.actions.map((action) => ({
                    ...action,
                    action: actions.find(({ name }) => name === action.name),
                    module:
                        action.module_name === 'this'
                            ? this.module
                            : modules.find((module) => module.info.name === action.module_name)
                }))
            )
        );
    }

    private clearDynamicDependencies(module: EntityModule) {
        const lookupDepInEventAction = (dep: ModelsDependencyItem) => {
            for (const eventName of Object.keys(module.current_event_config)) {
                const actions = module.current_event_config[eventName].actions.filter(
                    ({ module_name }) => module_name === dep.module_name
                );
                if (actions.length !== 0) {
                    return true;
                }
            }

            return false;
        };
        module.dynamic_dependencies.forEach((dep, index, list) => {
            if (dep.type !== DependencyType.ToMakeAction) {
                return;
            }
            if (dep.module_name === 'this' || !lookupDepInEventAction(dep)) {
                list.splice(index, 1);

                return;
            }
        });
    }
}
