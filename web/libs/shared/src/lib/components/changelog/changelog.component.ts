import { Component, EventEmitter, Input, Output } from '@angular/core';
import { ThemePalette } from '@ptsecurity/mosaic/core';
import { McSidepanelService } from '@ptsecurity/mosaic/sidepanel';

import { ModelsModuleS, ModelsModuleSShort } from '@soldr/api';

import { ModuleVersionPipe } from '../../pipes';
import { LanguageService } from '../../services';
import { EntityModule } from '../../types';

@Component({
    selector: 'soldr-changelog',
    templateUrl: './changelog.component.html',
    styleUrls: ['./changelog.component.scss'],
    providers: [ModuleVersionPipe]
})
export class ChangelogComponent {
    @Input() module: EntityModule | ModelsModuleS;
    @Input() versions: ModelsModuleSShort[];
    @Input() readOnly: boolean;
    @Input() selectable: boolean;

    @Output() install = new EventEmitter<string>();
    @Output() selectVersion = new EventEmitter<string>();

    language$ = this.languageService.current$;
    themePalette = ThemePalette;

    constructor(
        private languageService: LanguageService,
        private moduleVersionPipe: ModuleVersionPipe,
        private sidePanelService: McSidepanelService
    ) {}

    select(moduleVersion: ModelsModuleSShort) {
        if (this.selectable) {
            this.selectVersion.emit(this.moduleVersionPipe.transform(moduleVersion.info.version));
        } else {
            this.install.emit(this.moduleVersionPipe.transform(moduleVersion.info.version));
            this.sidePanelService.closeAll();
        }
    }
}
