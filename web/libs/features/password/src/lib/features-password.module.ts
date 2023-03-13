import { CommonModule } from '@angular/common';
import { NgModule } from '@angular/core';
import { ReactiveFormsModule } from '@angular/forms';
import { Route, RouterModule } from '@angular/router';

import { SharedModule } from '@soldr/shared';

import { PasswordChangeGuard } from './guards';
import { PasswordChangeComponent } from './routes';

export const featuresPasswordRoutes: Route[] = [
    {
        path: 'password_change',
        canActivate: [PasswordChangeGuard],
        component: PasswordChangeComponent
    }
];
@NgModule({
    imports: [CommonModule, RouterModule.forChild(featuresPasswordRoutes), SharedModule, ReactiveFormsModule],
    declarations: [PasswordChangeComponent],
    exports: [RouterModule]
})
export class FeaturesPasswordModule {}
