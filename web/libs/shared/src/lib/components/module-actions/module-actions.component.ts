import { Component, EventEmitter, Input, OnChanges, Output, SimpleChanges } from '@angular/core';
import { BehaviorSubject, combineLatest, map } from 'rxjs';

import { LanguageService } from '../../services';
import { EntityModule } from '../../types';
import { getActionParamsSchema } from '../../utils';
import { Sorting, SortingDirection } from '../grid/grid.types';

interface ActionItem {
    name: string;
    localizedName: string;
    localizedDescription: string;
    hasParams: boolean;
}

@Component({
    selector: 'soldr-module-actions',
    templateUrl: './module-actions.component.html',
    styleUrls: ['./module-actions.component.scss']
})
export class ModuleActionsComponent implements OnChanges {
    @Input() module: EntityModule;
    @Input() isReadOnly: boolean;

    @Output() saveModule = new EventEmitter<any>();

    actions$ = new BehaviorSubject<ActionItem[]>([]);
    actionsSorting$ = new BehaviorSubject<Sorting | Record<never, any>>({});
    sortedActions$ = combineLatest([this.actions$, this.actionsSorting$]).pipe(
        map(([items, sorting]: [ActionItem[], Sorting | Record<never, any>]) => [
            ...items.sort((a: ActionItem, b: ActionItem) => {
                if ((sorting as Sorting).order === SortingDirection.DESC) {
                    return b.localizedName.localeCompare(a.localizedName, 'en');
                } else {
                    return a.localizedName.localeCompare(b.localizedName, 'en');
                }
            })
        ])
    );

    constructor(private languageService: LanguageService) {}

    ngOnChanges({ module }: SimpleChanges): void {
        if (module?.currentValue) {
            const lang = this.languageService.lang;
            const schema = this.module.action_config_schema as Record<string, any>;
            const actions = Object.keys(schema.properties as object).map((actionName) => {
                const actionSchema = getActionParamsSchema(this.module, actionName);
                const locale = this.module.locale.actions[actionName][lang];
                const actionParams = Object.keys((actionSchema?.properties as object) || {});

                return {
                    name: actionName,
                    localizedName: locale.title,
                    localizedDescription: locale.description,
                    hasParams: actionParams.length > 0
                };
            });

            this.actions$.next(actions);
        }
    }

    saveActionConfig(module: EntityModule) {
        this.saveModule.emit(module);
    }
}
