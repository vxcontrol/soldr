import { ChangeDetectionStrategy, Component, Inject, OnDestroy, OnInit } from '@angular/core';
import { Event, NavigationEnd, NavigationStart, Router } from '@angular/router';
import { TranslocoService } from '@ngneat/transloco';
import { DateFormatter } from '@ptsecurity/mosaic/core';
import { DateTime } from 'luxon';
import { map, Observable, Subscription } from 'rxjs';

import { PublicInfo } from '@soldr/api';
import { PERMISSIONS_TOKEN, PermissionsService } from '@soldr/core';
import { LOCALES } from '@soldr/i18n';
import { ShortService } from '@soldr/models';
import { getFeaturePath, ProxyPermissionPage } from '@soldr/shared';
import { SharedFacade } from '@soldr/store/shared';

@Component({
    selector: 'soldr-navbar',
    templateUrl: './navbar.component.html',
    styleUrls: ['./navbar.component.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class NavbarComponent implements OnInit, OnDestroy {
    availableLocales: string[] = [];
    currentFeaturePath: string;
    info$!: Observable<PublicInfo>;
    isAuthorized$!: Observable<boolean>;
    locales = LOCALES;
    selectedServiceName$ = this.sharedFacade.selectedServiceName$;
    selectedServiceUrl$ = this.sharedFacade.selectedServiceUrl$;
    services$: Observable<ShortService[]> = this.sharedFacade.shortServices$;

    private subscription: Subscription = new Subscription();

    constructor(
        private dateFormatter: DateFormatter<DateTime>,
        private permissionsService: PermissionsService,
        private router: Router,
        private sharedFacade: SharedFacade,
        private translocoService: TranslocoService,
        @Inject(PERMISSIONS_TOKEN) public hasAccessTo: ProxyPermissionPage
    ) {}

    get currentLocale(): string {
        return this.translocoService.getActiveLang();
    }

    ngOnInit(): void {
        this.info$ = this.sharedFacade.selectInfo();
        this.isAuthorized$ = this.sharedFacade.selectIsAuthorized();

        const localesChangesSubscription = this.translocoService.langChanges$.subscribe(() => {
            this.availableLocales = (this.translocoService.getAvailableLangs() as string[]).sort();
        });
        this.subscription.add(localesChangesSubscription);

        const currentFeaturePathSubscription = this.router.events.subscribe((event: Event) => {
            if (event instanceof NavigationStart || event instanceof NavigationEnd) {
                this.currentFeaturePath = getFeaturePath(event.url);
            }
        });
        this.subscription.add(currentFeaturePathSubscription);
    }

    ngOnDestroy(): void {
        this.subscription.unsubscribe();
    }

    setLocale(locale: string): void {
        if (this.currentLocale !== locale) {
            const currentLang = locale.split('-')[0];
            this.dateFormatter.setLocale(currentLang);
            this.translocoService.setActiveLang(locale);
            // для firefox
            setTimeout(() => window.location.reload());
        }
    }

    logout(): void {
        this.sharedFacade.logout();
    }
}
