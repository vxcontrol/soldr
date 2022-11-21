import { AbstractControl, ValidationErrors, ValidatorFn } from '@angular/forms';

export function overlappingNamesValidator<T extends { name: string }>(): ValidatorFn {
    return (control: AbstractControl): ValidationErrors | null => {
        const items = (control?.parent?.parent?.value || []) as T[];
        const names = items.map(({ name }) => name);

        return names.filter((name) => (control.value as string) === name)?.length > 1 ? { overlapping: true } : null;
    };
}
