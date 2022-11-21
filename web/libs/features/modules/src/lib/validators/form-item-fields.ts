import { AbstractControl, ValidationErrors, ValidatorFn } from '@angular/forms';

export function formItemFieldsValidator(): ValidatorFn {
    return (control: AbstractControl): ValidationErrors | null => {
        let value;

        try {
            value = JSON.parse(control.value as string);
        } catch (e) {}

        return typeof value === 'object' && !Array.isArray(value) ? null : { formItemFields: true };
    };
}
