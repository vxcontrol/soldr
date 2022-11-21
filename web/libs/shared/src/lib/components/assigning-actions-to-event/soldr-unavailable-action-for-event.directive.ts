import { ChangeDetectorRef, Directive, ElementRef, Host, HostListener, Optional, Self } from '@angular/core';
import { ENTER, SPACE } from '@ptsecurity/cdk/keycodes';
import { McListOption } from '@ptsecurity/mosaic/list';

@Directive({
    selector: '[soldrUnavailableActionForEvent]'
})
export class SoldrUnavailableActionForEventDirective {
    constructor(
        @Host() @Self() @Optional() public host: McListOption,
        private elementRef: ElementRef,
        private changeDetector: ChangeDetectorRef
    ) {
        host.focus = () => {
            this.elementRef.nativeElement.focus();

            host.onFocus.next({ option: host });

            Promise.resolve().then(() => {
                host.hasFocus = true;

                this.changeDetector.markForCheck();
            });
        };

        const originalHandleClick = host.handleClick;

        host.handleClick = (event: MouseEvent) => {
            if (
                event
                    .composedPath()
                    .some((element) => (element as HTMLElement)?.tagName?.toLowerCase() === 'mc-pseudo-checkbox') &&
                !this.elementRef.nativeElement.classList.contains('action-item_disabled')
            ) {
                originalHandleClick.apply(this.host, [event]);
            } else {
                return;
            }
        };
    }

    @HostListener('keydown', ['$event'])
    keydown(event: KeyboardEvent) {
        if (
            (event.target as HTMLElement).classList.contains('action-item_disabled') &&
            [SPACE, ENTER].includes(event.keyCode)
        ) {
            event.stopPropagation();
        }
    }
}
