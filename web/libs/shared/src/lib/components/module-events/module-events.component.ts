import { Component, EventEmitter, Input, OnChanges, OnInit, Output, SimpleChanges } from '@angular/core';
import { TranslocoService } from '@ngneat/transloco';
import { BehaviorSubject, combineLatest, map, Observable, ReplaySubject } from 'rxjs';

import { ModelsEventConfigAction, ModelsOptionsActions } from '@soldr/api';
import { SharedFacade } from '@soldr/store/shared';

import { LanguageService, LocalizeActionService } from '../../services';
import { EntityModule, EventItem, LOG_TO_DB } from '../../types';
import { getEventParamsSchema, localizeSchemaParams } from '../../utils';
import { Sorting, SortingDirection } from '../grid/grid.types';

@Component({
    selector: 'soldr-module-events',
    templateUrl: './module-events.component.html',
    styleUrls: ['./module-events.component.scss']
})
export class ModuleEventsComponent implements OnInit, OnChanges {
    @Input() module: EntityModule;
    @Input() isReadOnly: boolean;
    @Input() inProgress: boolean;

    @Output() saveModuleEventConfig = new EventEmitter<EntityModule>();

    events$: Observable<EventItem[]>;
    eventsSearch$ = new BehaviorSubject('');
    eventsSearch: string;
    eventsSorting$ = new BehaviorSubject<Sorting | Record<never, any>>({});
    selectedEventName$ = new ReplaySubject<string>(1);
    selectedEvent$: Observable<EventItem>;
    module$ = new BehaviorSubject<EntityModule>(undefined);
    isLoading$ = combineLatest([this.module$, this.sharedFacade.isLoadingActions$]).pipe(
        map(([selectedModule, isLoadingActions]) => !selectedModule || isLoadingActions)
    );

    constructor(
        private languageService: LanguageService,
        private localizeActionService: LocalizeActionService,
        private sharedFacade: SharedFacade,
        private transloco: TranslocoService
    ) {
        this.sharedFacade.fetchActions();
    }

    ngOnInit(): void {
        this.defineEventsBlockObservables();
    }

    saveConfig(updatedModule: EntityModule) {
        this.saveModuleEventConfig.emit(updatedModule);
    }

    ngOnChanges({ module }: SimpleChanges): void {
        if (module?.currentValue) {
            this.module$.next(this.module);
        }
    }

    resetEvent(module: EntityModule, selectedEventName: string) {
        const updatedModule = {
            ...module,
            current_event_config: {
                ...module.current_event_config,
                [selectedEventName]: module.default_event_config[selectedEventName]
            }
        };

        this.saveModuleEventConfig.emit(updatedModule);
    }

    private defineEventsBlockObservables() {
        const localizeAction = (action: ModelsEventConfigAction, globalActions: ModelsOptionsActions[]) => {
            const globalAction = globalActions.find(({ name }) => action.name === name);
            const localization =
                action.name === LOG_TO_DB
                    ? {
                          localizedName: this.transloco.translate('shared.Shared.ModuleConfig.Text.LogToDbAction')
                      }
                    : {
                          localizedName: globalAction?.locale[this.languageService.lang].title || action.name,
                          localizedDescription: globalAction?.locale[this.languageService.lang].description || ''
                      };

            return {
                ...action,
                ...localization
            };
        };

        this.events$ = combineLatest([
            this.module$,
            this.eventsSearch$.pipe(map((v) => v || '')),
            this.eventsSorting$,
            this.sharedFacade.optionsActions$
        ]).pipe(
            map(
                ([module, searchValue, sorting, globalActions]: [
                    EntityModule,
                    string,
                    Sorting | Record<never, any>,
                    ModelsOptionsActions[]
                ]) =>
                    Object.keys((module?.event_config_schema.properties || {}) as object)
                        .map((name) => {
                            const event = module.current_event_config[name];
                            const actions = event.actions?.map((action) =>
                                this.localizeActionService.localizeAction(action, globalActions)
                            );
                            let schema = getEventParamsSchema(module, name);
                            const eventLocale = module.locale.events[name];

                            schema = localizeSchemaParams(
                                schema,
                                module.locale.event_config[name],
                                this.languageService.lang
                            );

                            const eventProps = Object.keys((schema?.properties as object) || {});

                            return {
                                actions,
                                hasParams: eventProps.length > 0,
                                localizedName: eventLocale?.[this.languageService.lang]?.title || name,
                                name,
                                schema
                            } as EventItem;
                        })
                        .filter(
                            (event: EventItem) =>
                                event.localizedName.toLowerCase().includes(searchValue.toLowerCase()) ||
                                event.name.toLowerCase().includes(searchValue.toLowerCase())
                        )
                        .sort((a: EventItem, b: EventItem) => {
                            if ((sorting as Sorting).order === SortingDirection.DESC) {
                                return b.localizedName.localeCompare(a.localizedName, 'en');
                            } else {
                                return a.localizedName.localeCompare(b.localizedName, 'en');
                            }
                        })
            )
        );

        this.selectedEvent$ = combineLatest([this.events$, this.selectedEventName$]).pipe(
            map(([events, selectedEventName]) => events.find(({ name }) => name === selectedEventName))
        );
    }
}
