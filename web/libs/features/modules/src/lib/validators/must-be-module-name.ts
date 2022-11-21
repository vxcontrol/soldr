import { AbstractControl, ValidationErrors, ValidatorFn } from '@angular/forms';

export function mustBeModuleNameValidator(): ValidatorFn {
    return (control: AbstractControl): ValidationErrors | null => {
        const allowed = !control.value || /^[a-z0-9_\-]+$/.test(control.value as string);

        return !allowed ? { mustBeModuleName: true } : null;
    };
}
