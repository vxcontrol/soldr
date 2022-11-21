import { Directive, AfterViewInit, ElementRef, Optional } from '@angular/core';
import { McCheckbox } from '@ptsecurity/mosaic/checkbox';

@Directive({
    selector: '[soldrAutofocus]'
})
export class AutofocusDirective implements AfterViewInit {
    constructor(private element: ElementRef, @Optional() private mcCheckbox: McCheckbox) {}

    ngAfterViewInit(): void {
        setTimeout(() => {
            const nativeElement: HTMLElement = this.element.nativeElement;
            // просто фокус на срабатывает на чекбоксе
            if (nativeElement.tagName === 'MC-CHECKBOX') {
                this.mcCheckbox.focus();
            } else {
                nativeElement.focus();
            }
        });
    }
}
