import { HttpErrorResponse } from '@angular/common/http';
import { Component, OnDestroy, OnInit } from '@angular/core';
import { FormControl, FormGroup } from '@angular/forms';
import { ActivatedRoute, Router } from '@angular/router';
import { TranslocoService } from '@ngneat/transloco';
import { ThemePalette } from '@ptsecurity/mosaic/core';
import { combineLatest, map, Subscription } from 'rxjs';

import { ErrorResponse, PublicService } from '@soldr/api';
import { PASSWORD_CHANGE_PAGE } from '@soldr/core';
import { LoginErrorCode, ModelsFormControl, ModelsFormGroup, PageTitleService } from '@soldr/shared';
import { SharedFacade } from '@soldr/store/shared';

interface LoginForm {
    mail: string;
    password: string;
}

@Component({
    selector: 'soldr-login-page',
    templateUrl: './login-page.component.html',
    styleUrls: ['./login-page.component.scss']
})
export class LoginPageComponent implements OnInit, OnDestroy {
    isSignInProcess = false;
    isPasswordChangeRequired = false;
    form!: ModelsFormGroup<LoginForm>;
    themePalette = ThemePalette;
    subscription = new Subscription();

    constructor(
        private activatedRoute: ActivatedRoute,
        private pageTitleService: PageTitleService,
        private publicService: PublicService,
        private router: Router,
        private sharedFacade: SharedFacade,
        private transloco: TranslocoService
    ) {}

    ngOnInit(): void {
        this.defineTitle();

        this.form = new FormGroup<ModelsFormControl<LoginForm>>({
            mail: new FormControl('', []),
            password: new FormControl('', [])
        });

        const passwordChangeSubscription = this.sharedFacade.isPasswordChangeRequired$.subscribe(
            (v) => (this.isPasswordChangeRequired = v)
        );
        this.subscription.add(passwordChangeSubscription);
    }

    ngOnDestroy(): void {
        this.subscription.unsubscribe();
    }

    login() {
        this.isSignInProcess = true;

        const data = this.form.getRawValue();

        this.publicService.login(data).subscribe({
            next: () => {
                this.isSignInProcess = false;
                if (this.isPasswordChangeRequired && this.nextUrl) {
                    this.router.navigate([PASSWORD_CHANGE_PAGE], { queryParams: { nextUrl: this.nextUrl } });
                } else {
                    this.router.navigateByUrl(
                        this.isPasswordChangeRequired ? PASSWORD_CHANGE_PAGE : this.urlAfterLogin
                    );
                }
            },
            error: (response: unknown) => {
                if (response instanceof HttpErrorResponse) {
                    const errorResponse: ErrorResponse = response.error as ErrorResponse;

                    if (errorResponse?.code === LoginErrorCode.InactiveUser) {
                        this.form.setErrors({
                            inactiveUser: true
                        });
                    } else if (errorResponse?.code === LoginErrorCode.InvalidCredentials) {
                        this.form.setErrors({
                            invalidCredentials: true
                        });
                    } else {
                        this.form.setErrors({
                            defaultError: true
                        });
                    }
                }

                this.isSignInProcess = false;
            }
        });
    }

    get urlAfterLogin() {
        return this.nextUrl || window.document.location.origin;
    }

    get nextUrl() {
        return this.activatedRoute.snapshot.queryParams.nextUrl as string;
    }

    private defineTitle() {
        const titlesSubscription = combineLatest([
            this.transloco.selectTranslate('Login.PageTitle.Text.Login', {}, 'login'),
            this.sharedFacade.selectedServiceName$,
            this.transloco.selectTranslate('Shared.Pseudo.PageTitle.ApplicationName', {}, 'shared')
        ])
            .pipe(map((segments) => segments.filter(Boolean)))
            .subscribe((segments) => this.pageTitleService.setTitle(segments));

        this.subscription.add(titlesSubscription);
    }
}
