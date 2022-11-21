import { Component, Input } from '@angular/core';
import { ThemePalette } from '@ptsecurity/mosaic/core';

import { ViewMode } from '../../types';

@Component({
    selector: 'soldr-dependency-status',
    templateUrl: './dependency-status.component.html',
    styleUrls: ['./dependency-status.component.scss']
})
export class DependencyStatusComponent {
    @Input() status: boolean;
    @Input() viewMode: ViewMode;

    themePalette = ThemePalette;
    viewModeEnum = ViewMode;

    constructor() {}
}
