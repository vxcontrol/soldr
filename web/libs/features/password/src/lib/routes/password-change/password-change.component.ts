import { Component, OnDestroy, OnInit, ViewChild } from '@angular/core';
import { ActivatedRoute, Router } from '@angular/router';
import { TranslocoService } from '@ngneat/transloco';
import { ThemePalette } from '@ptsecurity/mosaic/core';
import { combineLatest, map, pairwise, Subject, takeUntil, withLatestFrom } from 'rxjs';

import { LocationService } from '@soldr/core';
import { PageTitleService, PasswordFormComponent } from '@soldr/shared';
import { SharedFacade } from '@soldr/store/shared';

@Component({
    selector: 'soldr-password-change',
    templateUrl: './password-change.component.html',
    styleUrls: ['./password-change.component.scss']
})
export class PasswordChangeComponent implements OnInit, OnDestroy {
    @ViewChild('passwordFormComponent') passwordFormComponent: PasswordFormComponent;

    isChangingPassword$ = this.sharedFacade.isChangingPassword$;

    themePalette = ThemePalette;
    readonly destroyed$: Subject<void> = new Subject();

    constructor(
        private activatedRoute: ActivatedRoute,
        private locationService: LocationService,
        private pageTitleService: PageTitleService,
        private router: Router,
        private sharedFacade: SharedFacade,
        private transloco: TranslocoService
    ) {}

    get nextUrl() {
        return this.activatedRoute.snapshot.queryParams.nextUrl as string;
    }

    ngOnInit() {
        this.defineTitle();

        this.isChangingPassword$
            .pipe(pairwise(), withLatestFrom(this.sharedFacade.passwordChangeError$), takeUntil(this.destroyed$))
            .subscribe(([[prev, curr], changeError]) => {
                if (prev && !curr && !changeError) {
                    this.redirect();
                }
            });
    }

    ngOnDestroy() {
        this.destroyed$.next();
        this.destroyed$.complete();
    }

    save() {
        this.passwordFormComponent.passForm.ngSubmit.emit();
    }

    redirect() {
        if (this.nextUrl) {
            this.router.navigateByUrl(this.nextUrl);
        } else {
            this.locationService.redirectToFirstAvailablePage();
        }
    }

    private defineTitle() {
        combineLatest([
            this.transloco.selectTranslate('Password.PasswordPage.Title.PasswordChange', {}, 'password'),
            this.transloco.selectTranslate('Shared.Pseudo.PageTitle.ApplicationName', {}, 'shared')
        ])
            .pipe(
                map((segments) => segments.filter(Boolean)),
                takeUntil(this.destroyed$)
            )
            .subscribe((segments) => this.pageTitleService.setTitle(segments));
    }
}
