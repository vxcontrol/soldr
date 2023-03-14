import { Component, OnDestroy, OnInit, ViewChild } from '@angular/core';
import {
    AbstractControl,
    FormControl,
    FormGroup,
    NgForm,
    ValidationErrors,
    ValidatorFn,
    Validators
} from '@angular/forms';
import { ThemePalette } from '@ptsecurity/mosaic/core';
import { PasswordRules } from '@ptsecurity/mosaic/form-field';
import { first, pairwise, Subject, takeUntil, withLatestFrom } from 'rxjs';

import { ErrorResponse, PublicInfo, PublicService } from '@soldr/api';
import { SharedFacade } from '@soldr/store/shared';

import { REGEX_SPECIAL_SYMBOL } from '../../constants';
import { LoginErrorCode } from '../../types';

const MIN_PASS_LENGTH = 8;
const MAX_PASS_LENGTH = 50;
const REGEX_SYMBOL_TWICE = /^(?!.*(?<char>.).*\k<char>.*\k<char>).*$/;

@Component({
    selector: 'soldr-password-form',
    templateUrl: './password-form.component.html',
    styleUrls: ['./password-form.component.scss']
})
export class PasswordFormComponent implements OnInit, OnDestroy {
    @ViewChild('passForm') passForm: NgForm;
    isChangingPassword$ = this.sharedFacade.isChangingPassword$;

    minPassLength = MIN_PASS_LENGTH;
    maxPassLength = MAX_PASS_LENGTH;
    form: FormGroup<{
        current_password: FormControl<string>;
        password: FormControl<string>;
        confirm_password: FormControl<string>;
    }>;
    themePalette = ThemePalette;
    isPasswordChangeError = false;
    passwordRules = PasswordRules;
    specialSymbolRegex = REGEX_SPECIAL_SYMBOL;
    symbolTwiceRegex = REGEX_SYMBOL_TWICE;

    mail: string;
    readonly destroyed$: Subject<void> = new Subject();

    constructor(private publicService: PublicService, private sharedFacade: SharedFacade) {}

    get isFormEmpty() {
        return Object.values(this.form?.getRawValue() || {}).every((v) => !v);
    }

    get isSaveDisabled() {
        return this.form?.pristine || this.form?.invalid || this.isFormEmpty;
    }

    ngOnInit(): void {
        this.form = new FormGroup({
            current_password: new FormControl('', Validators.required),
            password: new FormControl('', [Validators.required, this.newPasswordValidator()]),
            confirm_password: new FormControl('', [Validators.required, this.confirmPasswordValidator()])
        });

        this.sharedFacade
            .selectInfo()
            .pipe(first())
            .subscribe((info: PublicInfo) => (this.mail = info.user?.mail));

        this.sharedFacade.isChangingPassword$
            .pipe(pairwise(), withLatestFrom(this.sharedFacade.passwordChangeError$), takeUntil(this.destroyed$))
            .subscribe(([[prev, curr], changeError]) => {
                if (prev && !curr && changeError) {
                    this.isPasswordChangeError = true;
                }
            });
    }

    ngOnDestroy(): void {
        this.destroyed$.next();
        this.destroyed$.complete();
    }

    save() {
        this.isPasswordChangeError = false;

        setTimeout(() => {
            if (this.form.invalid) {
                return;
            }

            const data = this.form.getRawValue();
            this.publicService
                .login({ mail: this.mail, password: data.current_password })
                .pipe(first())
                .subscribe({
                    next: () => this.sharedFacade.changePassword(data),
                    error: ({ error }: { error: ErrorResponse }) => {
                        if (error.code === LoginErrorCode.InvalidCredentials) {
                            this.form.controls.current_password.setErrors({ invalidPassword: true });
                        } else {
                            this.sharedFacade.changePassword(data);
                        }
                    }
                });
        });
    }

    private newPasswordValidator(): ValidatorFn {
        return (control: AbstractControl): ValidationErrors | null => {
            const currentPasswordControl = control.parent?.get('current_password');

            return control.value === currentPasswordControl?.value && currentPasswordControl?.valid
                ? { passwordEqualCurrent: true }
                : null;
        };
    }

    private confirmPasswordValidator(): ValidatorFn {
        return (control: AbstractControl): ValidationErrors | null => {
            const passwordControl = control.parent?.get('password');
            const currentPasswordControl = control.parent?.get('current_password');

            return control.value !== passwordControl?.value && passwordControl?.valid && currentPasswordControl?.valid
                ? { passwordNotEqual: true }
                : null;
        };
    }
}
