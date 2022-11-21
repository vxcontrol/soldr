import { Component, EventEmitter, Input, Output } from '@angular/core';
import { FormArray, FormGroup } from '@angular/forms';
import { ThemePalette } from '@ptsecurity/mosaic/core';

import { usedPropertyTypes } from '@soldr/shared';

@Component({
    selector: 'soldr-config-param-tabs',
    templateUrl: './config-param-tabs.component.html',
    styleUrls: ['./config-param-tabs.component.scss']
})
export class ConfigParamTabsComponent {
    @Input() activeTabIndex = 0;
    @Input() form: FormGroup;
    @Input() formArrayName: string;
    @Input() readOnly: boolean;

    @Output() deleteParam = new EventEmitter<string>();

    highlightedTabIndex = -1;
    propertiesTypes = usedPropertyTypes;
    themePalette = ThemePalette;

    constructor() {}

    get params(): FormArray {
        return this.form.get(this.formArrayName) as FormArray;
    }
}
