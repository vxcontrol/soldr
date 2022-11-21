// @ts-nocheck
import { get, set } from 'lodash-es';

export function replaceByProperties(object, paths, func) {
    const getPreparedKey = (v) => (v.includes('"') ? [v.replace(/"/g, '')] : v);

    for (const path of paths) {
        const preparedPath = Array.isArray(path) ? path.join('.') : path;
        const arr = preparedPath.matchAll(/(\"([\w\_\.]+)\"|([\w\_]+)|(\*))/gm);

        const [key, ...rest] = [...arr].map((item) => item[0]);

        if (key === '*') {
            // eslint-disable-next-line @typescript-eslint/no-unsafe-argument
            Object.keys(object).forEach(
                rest.length
                    ? (k) => replaceByProperties(get(object, getPreparedKey(k)), [rest.join('.')], func)
                    : (k) => set(object, [getPreparedKey(k)], func(get(object, getPreparedKey(k))))
            );
            continue;
        }

        const obj = get(object, getPreparedKey(key));

        if (!obj) {
            continue;
        }

        if (rest.length) {
            replaceByProperties(obj, [rest.join('.')], func);
        } else {
            // eslint-disable-next-line @typescript-eslint/no-unsafe-argument
            set(object, getPreparedKey(key), func(obj));
        }
    }
}
