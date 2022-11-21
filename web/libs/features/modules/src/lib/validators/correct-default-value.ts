import { AbstractControl, FormGroup, ValidationErrors, ValidatorFn } from '@angular/forms';

import { PropertyType } from '@soldr/shared';

export function correctDefaultValueValidator(fieldName = 'fields'): ValidatorFn {
    return (group: AbstractControl | FormGroup): ValidationErrors | null => {
        const fields = group.get(fieldName)?.value as string;
        const type = group.get('type')?.value as PropertyType;

        try {
            const defaultValue = JSON.parse(fields).default;

            if (defaultValue === undefined) {
                return null;
            }

            const defaultValueType = typeof defaultValue;
            let result = false;

            switch (type) {
                case 'boolean':
                case 'string':
                case 'number':
                    result = type === defaultValueType;
                    break;
                case 'integer':
                    result = Number.isInteger(defaultValue);
                    break;
                case 'object':
                    result = type === defaultValueType && !Array.isArray(defaultValue);
                    break;
                case 'array':
                    result = defaultValueType === 'object' && Array.isArray(defaultValue);
                    break;
                case 'none':
                    break;
                default:
                    result = false;
                    break;
            }

            return result ? null : { correctDefaultValue: true };
        } catch (e) {}

        return null;
    };
}
