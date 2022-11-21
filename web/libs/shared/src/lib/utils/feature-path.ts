const REGEX_FEATURE_PATH = /\/services\/[^\/]+\/(?<path>[a-z_-]+)/;

export const getFeaturePath = (url: string) => {
    const groups = REGEX_FEATURE_PATH.exec(url)?.groups as { path: string };

    return groups?.path || '';
};
