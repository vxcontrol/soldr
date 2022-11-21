import { Component, Input, OnInit } from '@angular/core';

@Component({
    selector: 'soldr-move-file-modal',
    templateUrl: './move-file-modal.component.html',
    styleUrls: ['./move-file-modal.component.scss']
})
export class MoveFileModalComponent implements OnInit {
    @Input() oldPath: string;
    @Input() prefix: string;

    path: string;

    constructor() {}

    ngOnInit(): void {
        this.path = this.oldPath;
    }
}
