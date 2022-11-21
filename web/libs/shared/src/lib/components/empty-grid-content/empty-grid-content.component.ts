import { Component, EventEmitter, Input, Output } from '@angular/core';
import { ThemePalette } from '@ptsecurity/mosaic/core';

@Component({
    selector: 'soldr-empty-grid-content',
    templateUrl: './empty-grid-content.component.html',
    styleUrls: ['./empty-grid-content.component.scss']
})
export class EmptyGridContentComponent {
    @Input() actionButtonText: string;
    @Input() description: string;
    @Input() isActionInProgress: boolean;
    @Input() isActionPermitted: boolean;
    @Input() title: string;

    @Output() emptyGridAction = new EventEmitter();

    themePalette = ThemePalette;

    constructor() {}
}
