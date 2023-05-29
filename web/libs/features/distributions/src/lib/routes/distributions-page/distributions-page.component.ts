import { Component, ElementRef, OnDestroy, OnInit, ViewChild } from '@angular/core';
import { TranslocoService } from '@ngneat/transloco';
import { combineLatest, combineLatestWith, map, pairwise, Subscription } from 'rxjs';

import { Architecture, Package, LanguageService, ModalInfoService, OperationSystem, PageTitleService } from '@soldr/shared';
import { SharedFacade } from '@soldr/store/shared';

@Component({
    selector: 'soldr-distributions-page',
    templateUrl: './distributions-page.component.html',
    styleUrls: ['./distributions-page.component.scss']
})
export class DistributionsPageComponent implements OnInit, OnDestroy {
    @ViewChild('firstButtonGroup') firstButtonGroup: ElementRef;

    agentBinaryVersions$ = this.sharedFacade.agentBinaryVersions$.pipe(
        map((binaryVersions) => binaryVersions)
    );
    isLoadingBinaries$ = this.sharedFacade.isLoadingAgentBinaries$;
    language$ = this.languageServe.current$;
    latestAgentBinary$ = this.sharedFacade.latestAgentBinary$;

    arch = Architecture;
    pack = Package;
    operationSystem = OperationSystem;
    subscription = new Subscription();

    constructor(
        private languageServe: LanguageService,
        private modalInfoService: ModalInfoService,
        private pageTitleService: PageTitleService,
        private sharedFacade: SharedFacade,
        private transloco: TranslocoService
    ) {}

    ngOnInit() {
        this.defineTitle();
        const loadingSubscription = this.isLoadingBinaries$.subscribe(() => {
            this.firstButtonGroup?.nativeElement.focus();
        });
        this.subscription.add(loadingSubscription);

        this.sharedFacade.fetchAgentBinaries();

        const isExportingBinarySubscription = this.sharedFacade.isExportingBinaryFile$
            .pipe(pairwise(), combineLatestWith(this.sharedFacade.exportError$))
            .subscribe(([[previous, current], exportError]) => {
                if (previous && !current && exportError) {
                    this.modalInfoService.openErrorInfoModal(
                        this.transloco.translate(
                            'distributions.Distributions.DistributionsPage.ErrorText.DownloadAgentError'
                        )
                    );
                }
            });
        this.subscription.add(isExportingBinarySubscription);
    }

    ngOnDestroy() {
        this.subscription.unsubscribe();
    }

    exportFile(os: OperationSystem, arch: Architecture, pack?: Package, version?: string) {
        this.sharedFacade.exportBinary(os, arch, pack, version);
    }

    private defineTitle() {
        const titlesSubscription = combineLatest([
            this.transloco.selectTranslate('Distributions.PageTitle.Text.Distributions', {}, 'distributions'),
            this.sharedFacade.selectedServiceName$,
            this.transloco.selectTranslate('Shared.Pseudo.PageTitle.ApplicationName', {}, 'shared')
        ])
            .pipe(map((segments) => segments.filter(Boolean)))
            .subscribe((segments) => this.pageTitleService.setTitle(segments));

        this.subscription.add(titlesSubscription);
    }
}
