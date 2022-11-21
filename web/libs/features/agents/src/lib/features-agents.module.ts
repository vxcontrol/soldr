import { CommonModule } from '@angular/common';
import { NgModule } from '@angular/core';
import { FormsModule, ReactiveFormsModule } from '@angular/forms';
import { RouterModule, Route } from '@angular/router';
import { AgGridModule } from 'ag-grid-angular';

import { FeaturesModulesInteractivityModule } from '@soldr/features/modules-interactivity';
import { ModulesGuard, SharedModule } from '@soldr/shared';

import { MoveToGroupComponent, EditAgentModalComponent } from './components';
import { DeleteAgentsModalComponent } from './components/delete-agents-modal/delete-agents-modal.component';
import { OsPipe } from './pipes';
import { AgentsPageComponent, AgentPageComponent, AgentModulePageComponent } from './routes';

export const featuresAgentsRoutes: Route[] = [
    {
        path: 'agents',
        component: AgentsPageComponent
    },
    {
        path: 'agents/:hash',
        component: AgentPageComponent
    },
    {
        path: 'agents/:hash/modules/:moduleName',
        canActivate: [ModulesGuard],
        data: { actions: ['view'] },
        component: AgentModulePageComponent
    }
];

@NgModule({
    imports: [
        CommonModule,
        RouterModule.forChild(featuresAgentsRoutes),
        SharedModule,
        FormsModule,
        ReactiveFormsModule,
        AgGridModule,
        FeaturesModulesInteractivityModule
    ],
    declarations: [
        AgentsPageComponent,
        AgentPageComponent,
        AgentModulePageComponent,
        MoveToGroupComponent,
        EditAgentModalComponent,
        DeleteAgentsModalComponent,
        OsPipe
    ]
})
export class FeaturesAgentsModule {}
