import { getChangesArrays } from '@soldr/shared';

export function applyChangesToObject(
    obj: Record<string, any>,
    arr1: string[],
    arr2: string[],
    callbackTransformer: (k: any) => any = null
): Record<string, any> {
    const [add, remove, rename] = getChangesArrays(arr1, arr2);

    add.forEach((k) => (obj[k] = callbackTransformer ? callbackTransformer(k) : {}));
    remove.forEach((k) => delete obj[k]);
    Object.entries(rename).forEach(([oldKey, newKey]) => {
        const getNewValue = () => {
            let resVal: any;

            if (obj[oldKey] !== undefined) return obj[oldKey];
            if (obj[newKey] !== undefined) return obj[newKey];
            if (callbackTransformer && (resVal = callbackTransformer(newKey)) !== undefined) return resVal;

            return {};
        };
        delete Object.assign(obj, { [newKey]: getNewValue() })[oldKey];
    });

    return obj;
}
