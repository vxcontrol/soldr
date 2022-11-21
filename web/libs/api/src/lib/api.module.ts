import { HttpClientModule } from '@angular/common/http';
import { Injector, NgModule } from '@angular/core';

import { LanguageService } from '@soldr/shared';

import {
    PublicService,
    AgentsService,
    GroupsService,
    TagsService,
    UpgradesService,
    VersionsService,
    PoliciesService,
    ModulesService,
    BinariesService,
    OptionsService
} from './services';

export let injectorInstance: Injector;

@NgModule({
    imports: [HttpClientModule],
    providers: [
        AgentsService,
        BinariesService,
        GroupsService,
        ModulesService,
        OptionsService,
        PoliciesService,
        PublicService,
        TagsService,
        UpgradesService,
        VersionsService,
        LanguageService
    ]
})
export class ApiModule {
    constructor(private injector: Injector) {
        injectorInstance = this.injector;
    }
}
