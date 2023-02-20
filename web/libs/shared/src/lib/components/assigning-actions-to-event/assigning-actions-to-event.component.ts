import { Component, EventEmitter, Input, OnInit, Output, TemplateRef, ViewChild } from '@angular/core';
import { TranslocoService } from '@ngneat/transloco';
import { ThemePalette } from '@ptsecurity/mosaic/core';
import { McListSelection, McListSelectionChange } from '@ptsecurity/mosaic/list';
import { McSidepanelService } from '@ptsecurity/mosaic/sidepanel';
import { BehaviorSubject, combineLatest, filter, map, Observable, shareReplay, Subject, take } from 'rxjs';

import {
    DependencyType,
    ModelsDependencyItem,
    ModelsModuleS,
    ModelsOptionsActions,
    ModelsOptionsFields
} from '@soldr/api';
import { LANGUAGES } from '@soldr/i18n';
import { SharedFacade } from '@soldr/store/shared';

import { LanguageService } from '../../services';
import { EntityModule, LOG_TO_DB, LOG_TO_DB_ACTION, THIS_MODULE_NAME } from '../../types';
import { clone, difference } from '../../utils';

interface ModuleListItem {
    label?: string;
    module: ModelsModuleS;
    actions?: ActionListItem[];
}

interface ActionListItem {
    action?: ModelsOptionsActions;
    module: ModelsModuleS;
    neededFields?: ModelsOptionsFields[];
    unavailable?: boolean;
}

@Component({
    selector: 'soldr-assigning-actions-to-event',
    templateUrl: './assigning-actions-to-event.component.html',
    styleUrls: ['./assigning-actions-to-event.component.scss']
})
export class AssigningActionsToEventComponent implements OnInit {
    @Input() module: EntityModule;

    @Output() saveModule = new EventEmitter<EntityModule>();

    @ViewChild('sidePanel') sidePanel: TemplateRef<any>;
    @ViewChild('list') list: McListSelection;

    actionsWithModulesTree$: Observable<ModuleListItem[]>;
    eventName: string;
    focusedAction$: Subject<ActionListItem> = new Subject<ActionListItem>();
    isLoading$ = combineLatest([this.sharedFacade.isLoadingAllModules$, this.sharedFacade.isLoadingActions$]).pipe(
        map((values) => values.some((v) => v))
    );
    language$ = this.languageService.current$;
    selected: string[];
    searchActions: string;
    searchActions$ = new BehaviorSubject('');
    themePalette = ThemePalette;

    private panelId = 'assigning-actions-to-event-panel';

    constructor(
        private languageService: LanguageService,
        private sharedFacade: SharedFacade,
        private sidePanelService: McSidepanelService,
        private transloco: TranslocoService
    ) {}

    ngOnInit(): void {
        this.defineObservables();
    }

    resetToDefault() {
        this.selected = (this.module.default_event_config[this.eventName]?.actions || []).map(({ name }) => name);
    }

    save() {
        this.sharedFacade.optionsActions$
            .pipe(shareReplay({ bufferSize: 1, refCount: false }), take(1))
            .subscribe((actions) => {
                const module = clone(this.module) as EntityModule;
                module.current_event_config[this.eventName].actions = this.selected.map((actionName) => {
                    const action = actions.find(({ name }) => name === actionName);

                    this.updateDynamicDependencies(module, actionName, action);

                    return actionName === LOG_TO_DB
                        ? LOG_TO_DB_ACTION
                        : {
                              name: actionName,
                              fields: action.config.fields,
                              priority: action.config.priority,
                              module_name: action.module_name
                          };
                });
                this.clearDynamicDependencies(module);
                this.saveModule.emit(module);
                this.sidePanelService.getSidepanelById(this.panelId).close();
            });
    }

    cancel() {
        this.sidePanelService.getSidepanelById(this.panelId).close();
    }

    open(eventName: string) {
        this.eventName = eventName;
        this.selected = this.module.current_event_config[eventName].actions.map(({ name }) => name);

        this.sharedFacade.fetchAllModules();
        this.sharedFacade.fetchActions();
        this.sharedFacade.fetchFields();

        this.sidePanelService.open(this.sidePanel, {
            id: this.panelId
        });

        this.actionsWithModulesTree$.pipe(take(1)).subscribe((tree) => {
            this.focusedAction$.next(tree[0]?.actions[0]);
        });

        this.sidePanelService
            .getSidepanelById(this.panelId)
            .afterClosed()
            .pipe(take(1))
            .subscribe(() => {
                this.searchActions = '';
                this.searchActions$.next('');
            });
    }

    changeSelected(event: McListSelectionChange) {
        this.selected = event.option.selected
            ? [...new Set([...(this.selected || []), event.option.value])]
            : this.selected.filter((value) => value !== event.option.value);
    }

    isSelected(actionName: string): boolean {
        return this.selected.includes(actionName);
    }

    private defineObservables() {
        this.actionsWithModulesTree$ = combineLatest([
            this.sharedFacade.allModules$,
            this.sharedFacade.optionsActions$,
            this.sharedFacade.optionsFields$,
            this.searchActions$
        ]).pipe(
            filter(([modules]) => !!modules.length),
            map(([modules, actions, fields, search]) => {
                const currentModule = modules.find((module) => this.module.info.name === module.info.name);
                const rest = modules.filter((module) => this.module.info.name !== module.info.name);
                const logging: ModuleListItem = this.getLoggingModuleListItem(currentModule, search);

                return [
                    logging,
                    ...rest
                        .filter((module) => Object.keys(module.default_action_config).length > 0)
                        .map(
                            (module) =>
                                ({
                                    module,
                                    actions: this.getActionListItems(module, actions, search, fields).sort(
                                        this.sortActionListItem(this.languageService.lang)
                                    )
                                } as ModuleListItem)
                        )
                        .sort(this.sortModuleListItems(this.languageService.lang))
                ];
            })
        );
    }

    private getActionListItems(
        module: ModelsModuleS,
        actions: ModelsOptionsActions[],
        search: string,
        fields: ModelsOptionsFields[]
    ) {
        return Object.keys(module.default_action_config)
            .filter((actionName) => {
                const action: ModelsOptionsActions = actions.find(({ name }) => name === actionName);

                return (
                    action.locale[this.languageService.lang].title.toLowerCase().includes(search.toLowerCase()) ||
                    action.name.toLowerCase().includes(search.toLowerCase())
                );
            })
            .map((actionName) => {
                const action: ModelsOptionsActions = actions.find(({ name }) => name === actionName);
                const neededFieldNames = difference(
                    this.module.default_event_config[this.eventName].fields,
                    action.config.fields
                );
                const neededFields = fields.filter(
                    ({ name, module_name }) => neededFieldNames.includes(name) && module_name === module.info.name
                );

                return {
                    action,
                    module,
                    unavailable: neededFieldNames.length > 0,
                    neededFields
                } as ActionListItem;
            });
    }

    private getLoggingModuleListItem(currentModule: ModelsModuleS, search: string): ModuleListItem {
        return {
            label: this.transloco.translate('shared.Shared.ModuleConfig.ListItemText.Logging'),
            module: currentModule,
            actions: [
                {
                    action: {
                        name: LOG_TO_DB_ACTION.name,
                        module_name: currentModule.info.name,
                        module_os: currentModule.info.os,
                        config: {
                            fields: LOG_TO_DB_ACTION.fields,
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
                        }
                    },
                    module: currentModule,
                    unavailable: false,
                    neededFields: []
                }
            ].filter(
                (item) =>
                    item.action.locale[this.languageService.lang].title.toLowerCase().includes(search.toLowerCase()) ||
                    item.action.name.toLowerCase().includes(search.toLowerCase())
            )
        };
    }

    private updateDynamicDependencies(module: EntityModule, actionName: string, action: ModelsOptionsActions) {
        if (actionName === LOG_TO_DB) {
            return;
        }
        if (!module.dynamic_dependencies.find((dependency) => dependency.module_name === action.module_name)) {
            module.dynamic_dependencies.push({
                module_name: action.module_name,
                type: DependencyType.ToMakeAction
            });
        }
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

        module.dynamic_dependencies = module.dynamic_dependencies.filter(
            (dep) =>
                dep.type !== DependencyType.ToMakeAction ||
                (dep.module_name !== THIS_MODULE_NAME && lookupDepInEventAction(dep))
        );
    }

    private sortModuleListItems(lang: string) {
        return (a: ActionListItem, b: ActionListItem) =>
            a.module.locale.module[lang].title.localeCompare(b.module.locale.module[lang].title, 'en');
    }

    private sortActionListItem(lang: string) {
        return (a: ActionListItem, b: ActionListItem) =>
            a.action.locale[lang].title.localeCompare(b.action.locale[lang].title, 'en');
    }
}
