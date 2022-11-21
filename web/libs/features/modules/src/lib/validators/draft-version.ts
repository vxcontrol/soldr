import { AbstractControl, ValidationErrors, ValidatorFn } from '@angular/forms';
import * as semver from 'semver';

export function draftVersion(currentVersion: string): ValidatorFn {
    return (control: AbstractControl): ValidationErrors | null => {
        const newVersion = control.value as string;
        const allowed = semver.valid(newVersion) && semver.gt(newVersion, currentVersion);

        return !allowed ? { draftVersion: true } : null;
    };
}
