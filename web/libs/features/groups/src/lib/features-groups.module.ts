import { CommonModule } from '@angular/common';
import { NgModule } from '@angular/core';
import { FormsModule, ReactiveFormsModule } from '@angular/forms';
import { Route, RouterModule } from '@angular/router';
import { AgGridModule } from 'ag-grid-angular';

import { FeaturesModulesInteractivityModule } from '@soldr/features/modules-interactivity';
import { LazyLoadTranslationsGuard } from '@soldr/i18n';
import { SharedModule, POLICY_LINKING_FACADE, ModulesGuard } from '@soldr/shared';

import { EditGroupModalComponent, DeleteGroupModalComponent } from './components';
import { GroupPageComponent, GroupsPageComponent, GroupModulePageComponent } from './routes';
import { LinkPolicyFacadeService } from './services';

export const featuresGroupsRoutes: Route[] = [
    {
        path: 'groups',
        component: GroupsPageComponent
    },
    {
        path: 'groups/:hash',
        canActivate: [LazyLoadTranslationsGuard],
        data: { scope: ['agents', 'policies'] },
        component: GroupPageComponent
    },
    {
        path: 'groups/:hash/modules/:moduleName',
        canActivate: [ModulesGuard],
        data: { actions: ['view'] },
        component: GroupModulePageComponent
    }
];

@NgModule({
    imports: [
        CommonModule,
        RouterModule.forChild(featuresGroupsRoutes),
        SharedModule,
        FormsModule,
        ReactiveFormsModule,
        AgGridModule,
        FeaturesModulesInteractivityModule
    ],
    declarations: [
        GroupPageComponent,
        GroupsPageComponent,
        GroupModulePageComponent,
        EditGroupModalComponent,
        DeleteGroupModalComponent
    ],
    providers: [{ provide: POLICY_LINKING_FACADE, useClass: LinkPolicyFacadeService }]
})
export class FeaturesGroupsModule {}
