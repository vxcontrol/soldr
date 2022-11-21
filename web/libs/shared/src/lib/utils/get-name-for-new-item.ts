export function getNameForNewItem(existedItems: string[], prefix: string) {
    const lastIndex =
        existedItems
            .filter((name) => new RegExp(`^${prefix}\\d+$`, 'm').test(name))
            .map((name) => +name.replace(prefix, ''))
            .sort((a, b) => b - a)[0] || 0;

    return `${prefix}${lastIndex + 1}`;
}
