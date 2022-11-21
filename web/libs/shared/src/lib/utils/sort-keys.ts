type A<B> = B | B[];

export function sortKeys<T>(x: A<T>): A<T> {
    if (typeof x !== 'object' || !x) return x;

    if (Array.isArray(x)) return x.map(sortKeys).sort() as T[];

    return Object.keys(x)
        .sort()
        .reduce((o, k) => ({ ...o, [k]: sortKeys(x[k as keyof T]) }), {}) as T;
}
