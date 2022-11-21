import { Component, Input } from '@angular/core';

@Component({
    selector: 'soldr-multiple-selection-panel',
    templateUrl: './multiple-selection-panel.component.html',
    styleUrls: ['./multiple-selection-panel.component.scss']
})
export class MultipleSelectionPanelComponent {
    @Input() title: string;
}
