import { AbstractControl, ValidationErrors, ValidatorFn } from '@angular/forms';

export function formItemNameValidator(allowDot?: boolean): ValidatorFn {
    return (control: AbstractControl): ValidationErrors | null => {
        const mask = allowDot ? new RegExp('^[.a-zA-Z0-9_]*$') : new RegExp('^[a-zA-Z0-9_]*$');
        const allowed = mask.test(control.value as string);

        return !allowed ? { formItemName: true } : null;
    };
}
