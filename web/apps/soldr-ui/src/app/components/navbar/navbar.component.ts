import { ChangeDetectionStrategy, Component, Inject, OnDestroy, OnInit, TemplateRef, ViewChild } from '@angular/core';
import { Event, NavigationEnd, NavigationStart, Router } from '@angular/router';
import { TranslocoService } from '@ngneat/transloco';
import { DateFormatter, ThemePalette } from '@ptsecurity/mosaic/core';
import { McModalRef, McModalService, ModalSize } from '@ptsecurity/mosaic/modal';
import { DateTime } from 'luxon';
import { Observable, pairwise, Subscription, withLatestFrom } from 'rxjs';

import { PublicInfo, PublicService, UserType } from '@soldr/api';
import { PERMISSIONS_TOKEN, PermissionsService } from '@soldr/core';
import { LOCALES } from '@soldr/i18n';
import { ShortService } from '@soldr/models';
import { getFeaturePath, PasswordFormComponent, ProxyPermissionPage } from '@soldr/shared';
import { SharedFacade } from '@soldr/store/shared';

@Component({
    selector: 'soldr-navbar',
    templateUrl: './navbar.component.html',
    styleUrls: ['./navbar.component.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class NavbarComponent implements OnInit, OnDestroy {
    @ViewChild('passwordModalFooter') passwordModalFooter: TemplateRef<any>;
    @ViewChild('passwordModalContent') passwordModalContent: TemplateRef<any>;
    @ViewChild('passwordFormComponent') passwordFormComponent: PasswordFormComponent;

    availableLocales: string[] = [];
    currentFeaturePath: string;
    info$!: Observable<PublicInfo>;
    isAuthorized$!: Observable<boolean>;
    isChangingPassword$ = this.sharedFacade.isChangingPassword$;
    locales = LOCALES;
    passwordModal: McModalRef;
    selectedServiceName$ = this.sharedFacade.selectedServiceName$;
    selectedServiceUrl$ = this.sharedFacade.selectedServiceUrl$;
    services$: Observable<ShortService[]> = this.sharedFacade.shortServices$;
    themePalette = ThemePalette;
    userType = UserType;

    private subscription: Subscription = new Subscription();

    constructor(
        private dateFormatter: DateFormatter<DateTime>,
        private modalService: McModalService,
        private permissionsService: PermissionsService,
        private publicService: PublicService,
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

        const changingPasswordSubscription = this.isChangingPassword$
            .pipe(pairwise(), withLatestFrom(this.sharedFacade.passwordChangeError$))
            .subscribe(([[prev, curr], changeError]) => {
                if (prev && !curr && !changeError) {
                    this.closePasswordModal();
                }
            });
        this.subscription.add(changingPasswordSubscription);
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

    openPasswordModal() {
        this.passwordModal = this.modalService.create({
            mcSize: ModalSize.Small,
            mcTitle: this.translocoService.translate('shared.Shared.PasswordForm.ModalTitle.PasswordChange'),
            mcContent: this.passwordModalContent,
            mcFooter: this.passwordModalFooter
        });
    }

    closePasswordModal() {
        this.passwordModal?.destroy();
    }
}
