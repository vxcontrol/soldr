import { AsyncValidatorFn, AbstractControl, ValidationErrors } from '@angular/forms';
import { catchError, map, Observable, of, take } from 'rxjs';

export function entityNameExistsValidator(list$: Observable<string[]>, exclude: string[] = []): AsyncValidatorFn {
    return (control: AbstractControl): Promise<ValidationErrors | null> | Observable<ValidationErrors | null> => {
        const value = (control.value as string).toLocaleLowerCase();

        return list$.pipe(
            take(1),
            map((list) =>
                list.find((item) => item.toLocaleLowerCase() === value) &&
                !exclude.map((value) => value.toLocaleLowerCase()).includes(value)
                    ? {
                          entityNameExists: true
                      }
                    : null
            ),
            catchError(() => of(null))
        );
    };
}
