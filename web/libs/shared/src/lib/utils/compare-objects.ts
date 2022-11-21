import { sortKeys } from './sort-keys';

export function compareObjects<T>(obj1: T, obj2: T) {
    return JSON.stringify(sortKeys(obj1)) === JSON.stringify(sortKeys(obj2));
}
