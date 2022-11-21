import MessageFormat from '@messageformat/core';

export class VueMessageFormatter {
    private locale: string;
    private formatter: MessageFormat;
    private caches: any;

    constructor(options: Record<string, any> = {}) {
        this.locale = options.locale || 'en';
        this.formatter = new MessageFormat(this.locale);
        this.caches = Object.create(null);
    }

    interpolate(message: string, values: Record<string, any>, path: string) {
        if (path === 'Shared.AgentModuleConfig.Text.ConfigReadOnly') {
            return '';
        }
        let fn = this.caches[message];
        if (!fn) {
            fn = this.formatter.compile(message);
            this.caches[message] = fn;
        }

        return [fn(values)];
    }
}
