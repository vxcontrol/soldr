import { GridApi, AgGridEvent } from 'ag-grid-community';
import { Observable } from 'rxjs';

export function gridEvent$<T extends AgGridEvent>(api: GridApi, eventType: string): Observable<T> {
    if (!api || !eventType) {
        return undefined;
    }

    return new Observable<T>((observer) => {
        const listener = (event: T) => {
            observer.next(event);
        };

        api.addEventListener(eventType, listener);

        return {
            unsubscribe: () => {
                api.removeEventListener(eventType, listener);
            }
        };
    });
}
