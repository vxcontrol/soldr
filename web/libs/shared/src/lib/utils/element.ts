import { Observable } from 'rxjs';

export function nodeMutation$(target: Node, options?: MutationObserverInit): Observable<MutationRecord[]> {
    return new Observable<MutationRecord[]>((observer) => {
        const mutationObserver = new MutationObserver((records) => {
            observer.next(records);
        });

        mutationObserver.observe(target, options);

        return {
            unsubscribe: () => mutationObserver.disconnect()
        };
    });
}

export function nodeResize$(target: Node): Observable<any[]> {
    return new Observable<any[]>((observer) => {
        const resizeObserver = new window.ResizeObserver((entries) => {
            observer.next(entries);
        });

        resizeObserver.observe(target as Element);

        return {
            unsubscribe: () => resizeObserver.disconnect()
        };
    });
}
