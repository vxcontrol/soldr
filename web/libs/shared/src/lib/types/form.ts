import { FormControl, FormGroup } from '@angular/forms';

export type ModelsFormGroup<T> = FormGroup<ModelsFormControl<T>>;

export type ModelsFormControl<T> = { [K in keyof T]: FormControl<T[K]> };
