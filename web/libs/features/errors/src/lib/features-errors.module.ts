import { CommonModule } from '@angular/common';
import { NgModule } from '@angular/core';
import { RouterModule, Routes } from '@angular/router';

import { SharedModule } from '@soldr/shared';

import { ForbiddenPageComponent } from './routes';

const routes: Routes = [
    {
        path: 'errors',
        component: ForbiddenPageComponent
    }
];

@NgModule({
    imports: [CommonModule, RouterModule.forChild(routes), SharedModule],
    declarations: [ForbiddenPageComponent]
})
export class FeaturesErrorsModule {}
