import { Component, EventEmitter, Inject, Input, OnChanges, OnInit, Output, SimpleChanges } from '@angular/core';
import { TranslocoService } from '@ngneat/transloco';
import { BehaviorSubject, map, Observable, combineLatest } from 'rxjs';

import { DependencyType } from '@soldr/api';
import { PERMISSIONS_TOKEN } from '@soldr/core';

import { LanguageService } from '../../services';
import { Dependency, ProxyPermission, ViewMode } from '../../types';

@Component({
    selector: 'soldr-dependencies-grid',
    templateUrl: './dependencies-grid.component.html',
    styleUrls: ['./dependencies-grid.component.scss']
})
export class DependenciesGridComponent implements OnInit, OnChanges {
    @Input() dependencies: Dependency[];
    @Input() viewMode: ViewMode;

    @Output() refresh = new EventEmitter();

    DependencyType = DependencyType;
    dependencies$ = new BehaviorSubject<Dependency[]>([]);
    language$ = this.languageService.current$;
    processedDependencies$: Observable<Dependency[]>;
    search$ = new BehaviorSubject<string>('');
    viewModeEnum = ViewMode;

    constructor(
        private transloco: TranslocoService,
        private languageService: LanguageService,
        @Inject(PERMISSIONS_TOKEN) public permitted: ProxyPermission
    ) {}

    ngOnInit(): void {
        this.processedDependencies$ = combineLatest([this.dependencies$, this.search$, this.language$]).pipe(
            map(([dependencies, search, language]) =>
                dependencies
                    .map((dependency) => ({
                        ...dependency,
                        description: this.transloco.translate(dependency.description, {
                            dependentObjectTitle:
                                dependency.module?.locale.module[language].title || dependency.moduleName,
                            sourceObjectTitle:
                                dependency.sourceModule?.locale.module[language].title || dependency.sourceModuleName,
                            sourceObjectOS: Object.keys(dependency.sourceModule?.info.os).join(', '),
                            minModuleVersion: dependency.minModuleVersion || 'empty',
                            minAgentVersion: dependency.minAgentVersion || 'empty'
                        })
                    }))
                    .filter((dependency) =>
                        [
                            dependency.module?.locale.module[language].title,
                            dependency.module?.locale.module[language].description,
                            dependency.sourceModule?.locale.module[language].title,
                            dependency.sourceModule?.locale.module[language].description,
                            this.viewMode !== this.viewModeEnum.Policies ? dependency.policy?.info.name[language] : '',
                            dependency.description
                        ]
                            .map((v) => v?.toLocaleLowerCase())
                            .some((v) => v?.includes(search?.toLocaleLowerCase() || ''))
                    )
                    .sort((a, b) => {
                        if (!a.status && b.status) {
                            return -1;
                        } else if (a.status && !b.status) {
                            return 1;
                        }

                        return a.sourceModule?.locale.module[language].title.localeCompare(
                            b.sourceModule?.locale.module[language].title,
                            'en'
                        );
                    })
            )
        );
    }

    ngOnChanges({ dependencies }: SimpleChanges): void {
        if (dependencies.currentValue) {
            this.dependencies$.next(this.dependencies);
        }
    }

    onSearch($event: string) {
        this.search$.next($event);
    }

    get searchPlaceholder() {
        return this.viewMode !== this.viewModeEnum.Policies
            ? this.transloco.translate('shared.Shared.DependenciesView.InputPlaceholder.SearchByFields')
            : this.transloco.translate('shared.Shared.DependenciesView.InputPlaceholder.SearchByFieldsForPolicy');
    }
}
