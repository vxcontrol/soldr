import { Component, ContentChild, Directive, Input } from '@angular/core';
import { ThemePalette } from '@ptsecurity/mosaic/core';

import { ProgressContainerColor, ProgressContainerOverlap } from '../../types';

@Directive({ selector: '[soldrProgressSpinner]' })
export class ProgressSpinnerDirective {}

@Component({
    selector: 'soldr-progress-container,[soldrProgressSpinner]',
    templateUrl: './progress-container.component.html',
    styleUrls: ['./progress-container.component.scss']
})
export class ProgressContainerComponent {
    themePalette = ThemePalette;
    progressContainerColor = ProgressContainerColor;
    progressContainerOverlap = ProgressContainerOverlap;

    @ContentChild(ProgressSpinnerDirective)
    progressSpinner: ContentChild;

    @Input()
    loadingFlag = false;

    @Input()
    color: ProgressContainerColor = ProgressContainerColor.Panel;

    @Input()
    text: string;

    @Input()
    overlap = ProgressContainerOverlap.UnderCdkOverlay;
}
