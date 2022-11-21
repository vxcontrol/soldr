import { ChangeDetectionStrategy, Component, EventEmitter, Inject, Input, Output } from '@angular/core';
import { ThemePalette } from '@ptsecurity/mosaic/core';

import { ModelsModuleSShort } from '@soldr/api';
import { PERMISSIONS_TOKEN } from '@soldr/core';
import { LanguageService, ModuleVersionPipe, ProxyPermission, sortTags, ViewMode } from '@soldr/shared';

import { EntityModule } from '../../types';

@Component({
    selector: 'soldr-module-info',
    templateUrl: './module-info.component.html',
    styleUrls: ['./module-info.component.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush,
    providers: [ModuleVersionPipe]
})
export class ModuleInfoComponent {
    @Input() module: EntityModule;
    @Input() versions?: ModelsModuleSShort[];
    @Input() viewMode: ViewMode;

    @Output() update = new EventEmitter<string>();
    @Output() seeVersions = new EventEmitter();

    language$ = this.languageService.current$;
    themePalette = ThemePalette;

    sortTags = sortTags;

    get canUpdate() {
        return (
            this.viewMode === ViewMode.Policies &&
            this.permitted.EditPolicies &&
            this.versions?.length > 0 &&
            this.versionPipe.transform(this.versions[0]?.info.version) !==
                this.versionPipe.transform(this.module.info.version)
        );
    }

    constructor(
        private languageService: LanguageService,
        private versionPipe: ModuleVersionPipe,
        @Inject(PERMISSIONS_TOKEN) public permitted: ProxyPermission
    ) {}

    getFiltrationByTag(tag: string) {
        return JSON.stringify([{ field: 'tags', value: [tag] }]);
    }

    doUpdate() {
        this.update.emit(this.versionPipe.transform(this.versions[0]?.info.version));
    }

    doSeeVersions() {
        this.seeVersions.emit();
    }
}
