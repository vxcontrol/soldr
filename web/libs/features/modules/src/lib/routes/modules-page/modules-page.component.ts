import { Component, Inject, OnDestroy, OnInit } from '@angular/core';
import { ActivatedRoute, Params, Router } from '@angular/router';
import { TranslocoService } from '@ngneat/transloco';
import { PopUpPlacements } from '@ptsecurity/mosaic/core';
import { combineLatest, filter, map, skipWhile, Subscription, take } from 'rxjs';

import { PERMISSIONS_TOKEN } from '@soldr/core';
import { Module } from '@soldr/models';
import {
    Filtration,
    GridColumnFilterItem,
    LanguageService,
    osList,
    PageTitleService,
    ProxyPermission,
    Sorting,
    sortTags
} from '@soldr/shared';
import { ModuleListFacade } from '@soldr/store/modules';
import { SharedFacade } from '@soldr/store/shared';

import { ModulesExporterService } from '../../services';

@Component({
    selector: 'soldr-modules-page',
    templateUrl: './modules-page.component.html',
    styleUrls: ['./modules-page.component.scss']
})
export class ModulesPageComponent implements OnInit, OnDestroy {
    gridFiltration$ = this.moduleListFacade.gridFiltration$;
    gridFiltrationByField$ = this.moduleListFacade.gridFiltrationByField$;
    isLoading$ = this.moduleListFacade.isLoading$;
    language$ = this.languageService.current$;
    modules$ = this.moduleListFacade.modules$;
    page$ = this.moduleListFacade.page$;
    placement = PopUpPlacements;
    searchString$ = this.moduleListFacade.search$;
    selected$ = this.moduleListFacade.selectedModules$.pipe(map((modules) => modules[0]));
    selectedModuleVersions$ = this.selected$.pipe(map((module) => Object.keys(module?.changelog || {})));
    sorting$ = this.moduleListFacade.sorting$;
    total$ = this.moduleListFacade.total$;
    info$ = this.sharedFacade.selectInfo();

    sortTags = sortTags;
    gridColumnsFilters: { [field: string]: GridColumnFilterItem[] } = { os: [...osList], tags: [] };
    subscription = new Subscription();

    constructor(
        private activatedRoute: ActivatedRoute,
        private languageService: LanguageService,
        private modulesExporter: ModulesExporterService,
        private moduleListFacade: ModuleListFacade,
        private pageTitleService: PageTitleService,
        private router: Router,
        private sharedFacade: SharedFacade,
        private transloco: TranslocoService,
        @Inject(PERMISSIONS_TOKEN) public permitted: ProxyPermission
    ) {}

    ngOnInit() {
        this.defineTitle();
        this.initPageStateRx();
        this.initDataRx();

        this.moduleListFacade.fetchModulesTags();

        const tagsSubscription = this.moduleListFacade.modulesTags$.subscribe((tags) => {
            this.gridColumnsFilters.tags = tags.map((tag) => ({ label: tag, value: tag }));
        });
        this.subscription.add(tagsSubscription);

        this.gridColumnsFilters.os = this.gridColumnsFilters.os.map((os) => ({
            ...os,
            label: this.transloco.translate(os.label)
        }));
    }

    ngOnDestroy(): void {
        this.moduleListFacade.reset();
        this.subscription.unsubscribe();
    }

    onGridSearch(value: string) {
        this.moduleListFacade.setGridSearch(value);
    }

    onGridSort(value: Sorting) {
        this.moduleListFacade.setGridSorting(value);
    }

    onGridSelectRows(modules: Module[]) {
        this.moduleListFacade.resetModuleErrors();
        this.moduleListFacade.selectModules(modules);
    }

    onResetFiltration() {
        this.moduleListFacade.resetFiltration();
    }

    onGridFilter(filtration: Filtration) {
        this.moduleListFacade.setGridFiltration(filtration);
    }

    loadNextPage(page: number) {
        this.moduleListFacade.fetchModulesPage(page);
    }

    setTag(tag: string) {
        this.moduleListFacade.setGridFiltrationByTag(tag);
    }

    refreshData() {
        this.moduleListFacade.fetchModulesPage(1);
    }

    afterImport() {
        this.refreshData();
    }

    onExport($event: { selected?: any[]; columns: string[] }) {
        this.modulesExporter.export($event.columns, $event.selected as Module[]);
    }

    private defineTitle() {
        const titlesSubscription = combineLatest([
            this.transloco.selectTranslate('Modules.PageTitle.Text.Modules', {}, 'modules'),
            this.sharedFacade.selectedServiceName$,
            this.transloco.selectTranslate('Shared.Pseudo.PageTitle.ApplicationName', {}, 'shared')
        ])
            .pipe(map((segments) => segments.filter(Boolean)))
            .subscribe((segments) => this.pageTitleService.setTitle(segments));

        this.subscription.add(titlesSubscription);
    }

    private initDataRx() {
        this.moduleListFacade.isRestored$.pipe(filter(Boolean), take(1)).subscribe(() => {
            this.refreshData();
        });
    }

    private initPageStateRx() {
        // restore state
        this.moduleListFacade.restoreState();

        // save state
        const saveStateSubscription = combineLatest([
            this.moduleListFacade.isRestored$,
            this.moduleListFacade.gridFiltration$,
            this.moduleListFacade.search$,
            this.moduleListFacade.sorting$
        ])
            .pipe(skipWhile(([restored]) => !restored))
            .subscribe(([, filtration, search, sorting]) => {
                const queryParams: Params = {
                    filtration: filtration.length > 0 ? JSON.stringify(filtration) : undefined,
                    search: search || undefined,
                    sort: Object.keys(sorting).length > 0 ? JSON.stringify(sorting) : undefined
                };

                this.router.navigate([], {
                    relativeTo: this.activatedRoute,
                    queryParams,
                    queryParamsHandling: 'merge',
                    replaceUrl: true
                });
            });
        this.subscription.add(saveStateSubscription);
    }
}
