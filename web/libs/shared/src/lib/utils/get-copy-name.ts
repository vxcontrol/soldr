export function getCopyName(originalName: string, existedNames: string[], copyPostfix: string): string {
    const hasCopyPattern = new RegExp(`^(.+)( \\(${copyPostfix}( (\\d+))?\\))$`, 'gm');

    // @ts-ignore
    const matches = [...originalName.matchAll(hasCopyPattern)][0];

    const base = (matches && matches[1]) || originalName;
    // eslint-disable-next-line @typescript-eslint/no-magic-numbers
    let nextIndex = (matches && matches[4]) || 0;
    let isAvailableName = false;
    let newName: string;
    do {
        newName = `${base} (${copyPostfix}${nextIndex === 0 ? '' : ` ${nextIndex}`})`;
        isAvailableName = existedNames.every((name) => name !== newName);
        nextIndex++;
    } while (!isAvailableName);

    return newName;
}
