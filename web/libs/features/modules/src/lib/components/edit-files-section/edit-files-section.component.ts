import { Component, Input, TemplateRef, ViewChild } from '@angular/core';
import { MC_TOAST_CONFIG, McToastPosition, McToastService } from '@ptsecurity/mosaic/toast';

import { ModelsModuleS } from '@soldr/api';

import { ModuleFolderSection } from '../../types';
import { EditFileModalComponent } from '../edit-file-modal/edit-file-modal.component';

@Component({
    selector: 'soldr-edit-files-section',
    templateUrl: './edit-files-section.component.html',
    styleUrls: ['./edit-files-section.component.scss'],
    providers: [
        McToastService,
        {
            provide: MC_TOAST_CONFIG,
            useValue: {
                position: McToastPosition.TOP_RIGHT,
                duration: 5000,
                delay: 2000,
                onTop: true
            }
        }
    ]
})
export class EditFilesSectionComponent {
    @Input() files: string[];
    @Input() module: ModelsModuleS;
    @Input() readOnly: boolean;

    @ViewChild('createFileBody') createFileBody: TemplateRef<any>;
    @ViewChild(EditFileModalComponent) editFileModal: EditFileModalComponent;

    agentTreeFilterText = '';
    serverTreeFilterText = '';
    browserTreeFilterText = '';
    ModuleFolderSection = ModuleFolderSection;

    constructor() {}

    openFile(filepath: string) {
        this.editFileModal.open(filepath);
    }
}
