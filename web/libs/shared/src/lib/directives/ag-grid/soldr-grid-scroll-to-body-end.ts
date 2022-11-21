import { Directive, EventEmitter, Output, ElementRef, OnDestroy } from '@angular/core';
import { AgGridAngular } from 'ag-grid-angular';
import { BodyScrollEvent, Events } from 'ag-grid-community';
import { Subscription } from 'rxjs';
import { debounceTime, filter, switchMap } from 'rxjs/operators';

import { gridEvent$ } from '../../utils';

const DEBOUNCE_SCROLL = 100;

@Directive({
    selector: '[soldrGridScrollToBodyEnd]'
})
export class SoldrGridScrollToBodyEndDirective implements OnDestroy {
    @Output('soldrGridScrollToBodyEnd') scrollToEnd = new EventEmitter<BodyScrollEvent>();

    private subscription = new Subscription();

    constructor(private hostAsElement: ElementRef, private host: AgGridAngular) {
        const gridReadySubscription = this.host.gridReady
            .pipe(
                switchMap(() => gridEvent$<BodyScrollEvent>(this.host.api, Events.EVENT_BODY_SCROLL)),
                filter((event) => event.direction === 'vertical'),
                debounceTime(DEBOUNCE_SCROLL)
            )
            .subscribe((event) => {
                const gridElement = this.hostAsElement?.nativeElement as Element;

                if (gridElement) {
                    const viewport = gridElement.querySelector('.ag-body-viewport');
                    const isScrolledToBottom =
                        Math.floor(viewport.scrollHeight - viewport.clientHeight - viewport.scrollTop) <= 0;

                    if (isScrolledToBottom) {
                        this.scrollToEnd.emit(event);
                    }
                }
            });

        this.subscription.add(gridReadySubscription);
    }

    ngOnDestroy(): void {
        this.subscription.unsubscribe();
    }
}
