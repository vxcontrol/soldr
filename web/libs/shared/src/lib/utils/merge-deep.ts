function isObject(item: any) {
    return item && typeof item === 'object' && !Array.isArray(item);
}

export function mergeDeep(target: any, ...sources: any[]): any {
    if (!sources.length) return target;
    const source = sources.shift();

    if (isObject(target) && isObject(source)) {
        for (const key in source) {
            if (isObject(source[key])) {
                if (!target[key]) Object.assign(target, { [key]: {} });
                mergeDeep(target[key], source[key]);
            } else {
                Object.assign(target, { [key]: source[key] });
            }
        }
    }

    // eslint-disable-next-line @typescript-eslint/no-unsafe-argument
    return mergeDeep(target, ...sources);
}
