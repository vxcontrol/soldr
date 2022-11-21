import { CommonModule } from '@angular/common';
import { NgModule } from '@angular/core';
import { FormsModule, ReactiveFormsModule } from '@angular/forms';
import { Route, RouterModule } from '@angular/router';

import { SharedModule } from '@soldr/shared';

import { LoginPageComponent } from './routes';

export const featuresLoginRoutes: Route[] = [
    {
        path: 'login',
        component: LoginPageComponent
    }
];

@NgModule({
    imports: [CommonModule, RouterModule.forChild(featuresLoginRoutes), FormsModule, ReactiveFormsModule, SharedModule],
    declarations: [LoginPageComponent],
    exports: [RouterModule]
})
export class FeaturesLoginModule {}
