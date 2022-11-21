import { CommonModule } from '@angular/common';
import { NgModule } from '@angular/core';
import { Route, RouterModule } from '@angular/router';

import { SharedModule } from '@soldr/shared';

import { DistributionsPageComponent } from './routes';

export const featuresDistributionsRoutes: Route[] = [
    {
        path: 'downloads',
        component: DistributionsPageComponent
    }
];

@NgModule({
    imports: [CommonModule, RouterModule.forChild(featuresDistributionsRoutes), SharedModule],
    declarations: [DistributionsPageComponent]
})
export class FeaturesDistributionsModule {}
