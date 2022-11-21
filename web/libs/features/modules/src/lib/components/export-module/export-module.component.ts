import { Component, Input, OnChanges, SimpleChanges, TemplateRef, ViewChild } from '@angular/core';
import { TranslocoService } from '@ngneat/transloco';
import { ThemePalette } from '@ptsecurity/mosaic/core';
import { McModalRef, McModalService, ModalSize } from '@ptsecurity/mosaic/modal';

import { ModelsModuleS, ModulesService } from '@soldr/api';
import { ModuleVersionPipe, saveFile } from '@soldr/shared';
import { ModuleListFacade } from '@soldr/store/modules';

@Component({
    selector: 'soldr-export-module',
    templateUrl: './export-module.component.html',
    styleUrls: ['./export-module.component.scss'],
    providers: [ModuleVersionPipe]
})
export class ExportModuleComponent implements OnChanges {
    @Input() selectedModule: ModelsModuleS;
    @ViewChild('tplFooter', { static: false }) tplFooter: TemplateRef<any>;

    isError = false;
    selectedVersion: string;
    themePalette = ThemePalette;
    tplModal: McModalRef;

    constructor(
        private modulesService: ModulesService,
        private modalService: McModalService,
        private moduleVersionPipe: ModuleVersionPipe,
        private modulesFacade: ModuleListFacade,
        private transloco: TranslocoService
    ) {}

    get hasMultipleVersions(): boolean {
        return Object.keys(this.selectedModule?.changelog || {}).length > 1;
    }

    ngOnChanges({ selectedModule }: SimpleChanges) {
        if (selectedModule.currentValue) {
            this.selectedVersion = this.moduleVersionPipe.transform(this.selectedModule.info.version);
        }
    }

    export(version?: string) {
        this.isError = false;

        this.modulesService.exportModule(this.selectedModule.info.name, version).subscribe({
            next: (response) => {
                const blob = new Blob([response.body], { type: 'application/zip' });
                const file = {
                    name: response.headers.get('content-disposition').split('filename=')[1],
                    url: window.URL.createObjectURL(blob)
                };
                saveFile(file);
            },
            error: () => {
                this.showErrorModal();
            }
        });
    }

    showErrorModal() {
        this.tplModal = this.modalService.create({
            mcSize: ModalSize.Small,
            mcContent: this.transloco.translate('modules.Modules.ExportModule.ErrorText.Internal'),
            mcFooter: this.tplFooter,
            mcClosable: false
        });
    }
}
