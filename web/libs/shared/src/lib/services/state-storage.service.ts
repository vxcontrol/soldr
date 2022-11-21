import { Injectable } from '@angular/core';

import { StateStorage } from '../directives';

@Injectable({
    providedIn: 'root'
})
export class StateStorageService implements StateStorage {
    private readonly localStorageKey = 'appState';
    private readonly delimiter = '.';
    private isLoaded = false;

    constructor() {}

    saveState(path: string, value: any): void {
        if (!this.isLoaded) {
            return;
        }
        const parts = path.split(this.delimiter);
        const savedValue = localStorage.getItem(this.localStorageKey);
        const savedState = savedValue ? JSON.parse(savedValue) : {};

        parts.reduce((acc, part, index) => {
            if (!acc[part] && index !== parts.length - 1) {
                acc[part] = {};
            }

            if (index === parts.length - 1) {
                acc[part] = value;
            }

            return acc[part];
        }, savedState);

        localStorage.setItem(this.localStorageKey, JSON.stringify(savedState));
    }

    loadState(path?: string): Record<string, any> {
        const savedValue = localStorage.getItem(this.localStorageKey);

        this.isLoaded = true;

        const savedState = savedValue ? JSON.parse(savedValue) : {};
        let value = savedState;

        if (path) {
            const parts = path.split(this.delimiter);

            value = parts.reduce((acc, part) => {
                if (!acc || acc[part] === undefined) {
                    return undefined;
                }

                return acc[part];
            }, savedState);
        }

        return value;
    }
}
