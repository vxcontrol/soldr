import { ModelsSemVersion } from '@soldr/api';

export function convertVersion(value = ''): ModelsSemVersion {
    const numbers = value
        .trim()
        .replace(/^v?/g, '')
        .split('.')
        .map((item) => +item);

    return {
        major: numbers[0] || 0,
        minor: numbers[1] || 0,
        patch: numbers[2] || 0
    };
}
