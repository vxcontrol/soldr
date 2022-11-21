import { AbstractControl, ValidationErrors, ValidatorFn } from '@angular/forms';

export function mustBeModuleVersionValidator(): ValidatorFn {
    return (control: AbstractControl): ValidationErrors | null => {
        const allowed = !control.value || /^\d+.\d+.\d+$/.test(control.value as string);

        return !allowed ? { mustBeModuleVersion: true } : null;
    };
}
