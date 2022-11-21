import { Component, Input } from '@angular/core';
import { ThemePalette } from '@ptsecurity/mosaic/core';

import { ViewMode } from '../../types';

@Component({
    selector: 'soldr-consistency-icon',
    templateUrl: './consistency-icon.component.html',
    styleUrls: ['./consistency-icon.component.scss']
})
export class ConsistencyIconComponent {
    @Input() viewMode: ViewMode;
    @Input() isModuleInstance = false;

    themePalette = ThemePalette;
    viewModeEnum = ViewMode;

    constructor() {}
}
