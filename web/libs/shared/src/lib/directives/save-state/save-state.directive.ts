import { Directive, Inject, InjectionToken, Input, OnChanges, SimpleChanges } from '@angular/core';

import { StateStorageService } from '@soldr/shared';

export abstract class StateStorage {
    abstract loadState(path?: string): Record<string, any>;

    abstract saveState(path: string, value: any): void;
}

export const STATE_STORAGE_TOKEN: InjectionToken<StateStorage> = new InjectionToken<StateStorage>('STATE_STORAGE', {
    providedIn: 'root',
    factory: () => new StateStorageService()
});

@Directive({
    selector: '[soldrSaveState]'
})
export class SaveStateDirective implements OnChanges {
    @Input() saveStateKey: string;
    @Input() saveStateValue: any;

    constructor(@Inject(STATE_STORAGE_TOKEN) private stateStorage: StateStorage) {}

    ngOnChanges({ saveStateKey, saveStateValue }: SimpleChanges): void {
        if ((saveStateKey?.currentValue || saveStateValue?.currentValue !== undefined) && this.saveStateKey) {
            this.stateStorage.saveState(this.saveStateKey, this.saveStateValue);
        }
    }
}
