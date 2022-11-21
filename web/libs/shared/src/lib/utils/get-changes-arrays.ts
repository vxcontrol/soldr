export type ChangesArrays = [string[], string[], Record<string, any>];

export function getChangesArrays(a: string[], b: string[]): ChangesArrays {
    const arr1 = [...a].sort();
    const arr2 = [...b].sort();

    const diff1 = arr1.filter((x) => !arr2.includes(x));
    const diff2 = arr2.filter((x) => !arr1.includes(x));
    const diff = [].concat(diff1).concat(diff2) as string[];
    if (diff.length === 0) {
        return [[], [], {}];
    }
    if (diff1.length === diff2.length) {
        return [
            [],
            [],
            diff1.reduce(
                (acc, k, i) => ({
                    ...acc,
                    [k]: diff2[i]
                }),
                {}
            )
        ];
    }

    return [diff2, diff1, {}];
}
