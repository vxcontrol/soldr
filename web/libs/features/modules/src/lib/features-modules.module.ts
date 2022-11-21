import { CommonModule } from '@angular/common';
import { NgModule } from '@angular/core';
import { FormsModule, ReactiveFormsModule } from '@angular/forms';
import { Route, RouterModule } from '@angular/router';
import { ReactiveComponentModule } from '@ngrx/component';

import { CanLeavePageGuard, ModulesGuard, SharedModule } from '@soldr/shared';

import {
    ConfigParamTabsComponent,
    CreateFileModalComponent,
    CreateModuleDraftModalComponent,
    CreateModuleModalComponent,
    DeleteFileModalComponent,
    DeleteModuleComponent,
    EditActionsSectionComponent,
    EditConfigSectionComponent,
    EditDependenciesSectionComponent,
    EditEventsSectionComponent,
    EditFieldsSectionComponent,
    EditFileModalComponent,
    EditFilesSectionComponent,
    EditGeneralSectionComponent,
    EditLocalizationSectionComponent,
    EditSecureConfigSectionComponent,
    ExportModuleComponent,
    FilesTreeComponent,
    ImportModuleModalComponent,
    MoveFileModalComponent,
    ReleaseModuleModalComponent,
    ToggleItemComponent,
    ToggleListComponent
} from './components';
import { EditChangelogSectionComponent } from './components/edit-changelog-section/edit-changelog-section.component';
import { ModulesPageComponent, EditModulePageComponent } from './routes';

export const featuresModulesRoutes: Route[] = [
    {
        path: 'modules',
        component: ModulesPageComponent
    },
    {
        path: 'modules/:name/edit',
        component: EditModulePageComponent,
        canActivate: [ModulesGuard],
        data: { actions: ['edit'] },
        canDeactivate: [CanLeavePageGuard]
    }
];

@NgModule({
    imports: [
        CommonModule,
        RouterModule.forChild(featuresModulesRoutes),
        SharedModule,
        ReactiveFormsModule,
        FormsModule,
        ReactiveComponentModule
    ],
    declarations: [
        ConfigParamTabsComponent,
        CreateFileModalComponent,
        CreateModuleModalComponent,
        DeleteFileModalComponent,
        DeleteModuleComponent,
        EditActionsSectionComponent,
        EditConfigSectionComponent,
        EditEventsSectionComponent,
        EditFieldsSectionComponent,
        EditFileModalComponent,
        EditFilesSectionComponent,
        EditGeneralSectionComponent,
        EditLocalizationSectionComponent,
        EditModulePageComponent,
        EditSecureConfigSectionComponent,
        ExportModuleComponent,
        FilesTreeComponent,
        ImportModuleModalComponent,
        ModulesPageComponent,
        MoveFileModalComponent,
        ToggleItemComponent,
        ToggleListComponent,
        EditDependenciesSectionComponent,
        EditChangelogSectionComponent,
        ReleaseModuleModalComponent,
        CreateModuleDraftModalComponent
    ]
})
export class FeaturesModulesModule {}
