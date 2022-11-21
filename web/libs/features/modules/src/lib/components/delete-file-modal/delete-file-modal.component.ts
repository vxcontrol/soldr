import { Component, Input } from '@angular/core';

@Component({
    selector: 'soldr-delete-file-modal',
    templateUrl: './delete-file-modal.component.html',
    styleUrls: ['./delete-file-modal.component.scss']
})
export class DeleteFileModalComponent {
    @Input() path: string;

    constructor() {}
}
