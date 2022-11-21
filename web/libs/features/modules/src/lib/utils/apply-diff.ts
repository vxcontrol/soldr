import { FormArray, FormGroup } from '@angular/forms';

export function applyDiff<T extends Record<string, any>>(
    formArray: FormArray,
    [added, removed, renamed, changed]: [string[], string[], Record<string, string>, string[]?],
    oldModel: T[],
    newModel: T[],
    callback: (param: T) => FormGroup,
    keyName = 'name'
) {
    const indexesForRemoving = removed
        .map((removedKey) => oldModel.findIndex((item: T) => item[keyName] === removedKey))
        .sort((a, b) => b - a);

    for (const removedIndex of indexesForRemoving) {
        formArray.removeAt(removedIndex, { emitEvent: false });
    }

    for (const addedKey of added) {
        const param = newModel.find((item: T) => item[keyName] === addedKey);
        formArray.push(callback(param), { emitEvent: false });
    }

    for (const changedKey of changed || []) {
        const index = newModel.findIndex((item: T) => item[keyName] === changedKey);
        const param = newModel[index];
        formArray.setControl(index, callback(param));
    }
}
