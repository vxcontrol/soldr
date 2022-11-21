import { CommonModule } from '@angular/common';
import { NgModule } from '@angular/core';
import { FormsModule, ReactiveFormsModule } from '@angular/forms';
import { RouterModule, Route } from '@angular/router';
import { ReactiveComponentModule } from '@ngrx/component';
import { AgGridModule } from 'ag-grid-angular';

import { FeaturesModulesInteractivityModule } from '@soldr/features/modules-interactivity';
import { LazyLoadTranslationsGuard } from '@soldr/i18n';
import { POLICY_LINKING_FACADE, SharedModule } from '@soldr/shared';

import { DeletePolicyModalComponent, EditPolicyModalComponent } from './components';
import { OsListPipe } from './pipes';
import { PolicyPageComponent, PoliciesPageComponent, PolicyModulePageComponent } from './routes';
import { LinkPolicyFacadeService } from './services';

export const featuresPoliciesRoutes: Route[] = [
    {
        path: 'policies',
        component: PoliciesPageComponent
    },
    {
        path: 'policies/:hash',
        canActivate: [LazyLoadTranslationsGuard],
        data: { scope: ['agents', 'groups'] },
        component: PolicyPageComponent
    },
    {
        path: 'policies/:hash/modules/:moduleName',
        component: PolicyModulePageComponent
    }
];

@NgModule({
    imports: [
        CommonModule,
        RouterModule.forChild(featuresPoliciesRoutes),
        SharedModule,
        FormsModule,
        ReactiveFormsModule,
        ReactiveComponentModule,
        AgGridModule,
        FeaturesModulesInteractivityModule
    ],
    declarations: [
        DeletePolicyModalComponent,
        EditPolicyModalComponent,
        OsListPipe,
        PoliciesPageComponent,
        PolicyModulePageComponent,
        PolicyPageComponent
    ],
    providers: [{ provide: POLICY_LINKING_FACADE, useClass: LinkPolicyFacadeService }]
})
export class FeaturesPoliciesModule {}
