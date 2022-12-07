import { ContentObserver } from '@angular/cdk/observers';
import {
    AfterViewInit,
    ChangeDetectorRef,
    Component,
    Directive,
    ElementRef,
    Inject,
    Input,
    OnDestroy
} from '@angular/core';
import { PopUpPlacements } from '@ptsecurity/mosaic/core';
import { Subscription } from 'rxjs';

import { MosaicTokens, THEME_TOKENS } from '@soldr/core';

@Directive({ selector: '[soldrTextOverflow]' })
export class TextOverflowDirective {}

@Component({
    selector: 'soldr-text-overflow,[soldrTextOverflow]',
    template: `
        <div
            title="{{ hasTitle ? text : '' }}"
            class="soldr-text-overflow ellipsis"
            [mcTooltip]="text"
            [mcPlacement]="tooltipPlacement"
            [mcTooltipDisabled]="!isTooltip || hasTitle"
        >
            <ng-content></ng-content>
        </div>
    `,
    styles: [
        `
            :host {
                display: flex;
                align-items: center;
                width: 100%;
                height: 100%;
            }
        `
    ]
})
export class TextOverflowComponent implements AfterViewInit, OnDestroy {
    @Input() tooltipPlacement: PopUpPlacements = PopUpPlacements.TopLeft;

    isTooltip: boolean;
    observer: ResizeObserver;
    text: string;
    hasTitle: boolean;
    element: Element;
    tooltipContentWidth = parseInt(this.tokens.TooltipSizeMaxWidth, 10) - 2 * parseInt(this.tokens.TooltipSizePadding);

    private subscription: Subscription = new Subscription();

    constructor(
        public elementRef: ElementRef,
        private contentObserver: ContentObserver,
        private changeDetectorRef: ChangeDetectorRef,
        @Inject(THEME_TOKENS) public tokens: MosaicTokens
    ) {}

    ngAfterViewInit() {
        setTimeout(() => {
            this.onChangeText();
            this.subscription = this.contentObserver.observe(this.elementRef.nativeElement as Element).subscribe(() => {
                this.onChangeText();
            });

            this.observer = new ResizeObserver((entries) => {
                entries.forEach(
                    () => (this.isTooltip = this.element.scrollWidth > (this.element as HTMLElement).offsetWidth)
                );
                this.changeDetectorRef.detectChanges();
            });

            this.observer.observe(this.element);
        });
    }

    ngOnDestroy() {
        this.observer?.disconnect();
        this.subscription.unsubscribe();
    }

    onChangeText() {
        this.hasTitle = false;
        this.element = this.elementRef.nativeElement as Element;
        this.findLastChild();
        this.text = this.element.textContent;
        if (this.element.clientWidth > this.tooltipContentWidth) {
            this.hasTitle = true;
        }
        this.changeDetectorRef.detectChanges();
    }

    findLastChild() {
        while (this.element.firstElementChild) {
            this.element = this.element.firstElementChild;
        }
    }
}
