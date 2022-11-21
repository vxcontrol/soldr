import { Component, Inject, Input, OnChanges, SimpleChanges } from '@angular/core';
import { BehaviorSubject, filter, from, map, Subject, switchMap, toArray, combineLatest } from 'rxjs';

import { DependencyType, ModelsModuleS } from '@soldr/api';
import { PERMISSIONS_TOKEN } from '@soldr/core';

import { LanguageService } from '../../services';
import { EntityModule, ProxyPermission } from '../../types';

@Component({
    selector: 'soldr-dependencies-info',
    templateUrl: './dependencies-info.component.html',
    styleUrls: ['./dependencies-info.component.scss']
})
export class DependenciesInfoComponent implements OnChanges {
    @Input() module: EntityModule | ModelsModuleS;
    @Input() modules: ModelsModuleS[];
    @Input() loading: boolean;

    module$ = new Subject<EntityModule | ModelsModuleS>();
    modulesByName: Record<string, ModelsModuleS> = {};
    language$ = this.languageService.current$;

    dependencies$ = this.module$.pipe(
        map((module) => [...((module as EntityModule).dynamic_dependencies || []), ...module.static_dependencies])
    );
    agentDependency$ = this.dependencies$.pipe(
        switchMap((dependencies) =>
            from(dependencies).pipe(filter((dependency) => dependency.type === DependencyType.AgentVersion))
        )
    );
    receiveDataDependencies$ = this.dependencies$.pipe(
        switchMap((dependencies) =>
            from(dependencies).pipe(
                filter((dependency) => dependency.type === DependencyType.ToReceiveData),
                toArray()
            )
        )
    );
    sendDataDependencies$ = this.dependencies$.pipe(
        switchMap((dependencies) =>
            from(dependencies).pipe(
                filter((dependency) => dependency.type === DependencyType.ToSendData),
                toArray()
            )
        )
    );
    loading$ = new BehaviorSubject<boolean>(false);
    isEmpty$ = combineLatest([this.loading$, this.dependencies$]).pipe(
        map(([loading, deps]) => loading || deps?.length === 0)
    );

    constructor(
        private languageService: LanguageService,
        @Inject(PERMISSIONS_TOKEN) public permitted: ProxyPermission
    ) {}

    ngOnChanges({ module, modules, loading }: SimpleChanges): void {
        if (module?.currentValue) {
            this.module$.next(this.module);
        }

        if (modules?.currentValue) {
            this.modulesByName = this.modules.reduce((acc, module) => ({ ...acc, [module.info.name]: module }), {});
        }

        if (loading) {
            this.loading$.next(this.loading);
        }
    }
}
