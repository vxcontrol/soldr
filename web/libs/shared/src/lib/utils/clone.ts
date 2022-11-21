export function clone(value: any) {
    try {
        return JSON.parse(JSON.stringify(value));
    } catch {
        return undefined;
    }
}
