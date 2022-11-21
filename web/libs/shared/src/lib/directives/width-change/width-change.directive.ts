import { ElementRef, Directive, Output, EventEmitter, OnDestroy } from '@angular/core';
import { Subscription } from 'rxjs';
import { debounceTime } from 'rxjs/operators';

import { nodeMutation$ } from '../../utils/element';

export interface WidthChangeEvent {
    oldWidth: string;
    width: string;
}

export const WIDTH_CHANGE_MUTATION_TIMEOUT = 300;

@Directive({
    selector: '[soldrWidthChange]'
})
export class WidthChangeDirective implements OnDestroy {
    @Output() soldrWidthChange = new EventEmitter<WidthChangeEvent>();

    private lastWidth: string;
    private subscription = new Subscription();

    constructor(private element: ElementRef) {
        this.lastWidth = this.formatNodeWidth();

        const subscription = nodeMutation$(this.element.nativeElement as Node, { attributes: true })
            .pipe(debounceTime(WIDTH_CHANGE_MUTATION_TIMEOUT))
            .subscribe(() => {
                const currentWidth = this.formatNodeWidth();

                if (this.lastWidth !== currentWidth) {
                    this.soldrWidthChange.emit({
                        oldWidth: this.lastWidth,
                        width: currentWidth
                    });
                }

                this.lastWidth = currentWidth;
            });

        this.subscription.add(subscription);
    }

    private formatNodeWidth(): string {
        const element = this.element.nativeElement;
        const rect = element?.getBoundingClientRect();

        return rect?.width ? `${rect.width}px` : '0px';
    }

    ngOnDestroy(): void {
        this.subscription.unsubscribe();
    }
}
