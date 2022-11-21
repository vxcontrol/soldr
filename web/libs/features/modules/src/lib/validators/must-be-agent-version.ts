import { AbstractControl, ValidationErrors, ValidatorFn } from '@angular/forms';

export function mustBeAgentVersionValidator(): ValidatorFn {
    return (control: AbstractControl): ValidationErrors | null => {
        const allowed = !control.value || /^v?\d+.\d+.\d(.\d+)?(\-[\d\w]+)?$/.test(control.value as string);

        return !allowed ? { mustBeAgentVersion: true } : null;
    };
}
