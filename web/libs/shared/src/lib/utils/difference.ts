export const difference = (arr1: string[], arr2: string[]) => {
    const set1 = new Set(arr1);
    const set2 = new Set(arr2);
    const intersect = new Set([...set1].filter((x) => set2.has(x)));

    return [...set2].filter((x) => !intersect.has(x));
};
