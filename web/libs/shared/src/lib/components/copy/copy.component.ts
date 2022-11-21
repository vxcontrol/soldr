import { Component, Input } from '@angular/core';

@Component({
    selector: 'soldr-copy',
    templateUrl: './copy.component.html',
    styleUrls: ['./copy.component.scss']
})
export class CopyComponent {
    @Input() value: string;

    constructor() {}

    copy() {
        window.navigator.clipboard?.writeText(this.value);
    }
}
