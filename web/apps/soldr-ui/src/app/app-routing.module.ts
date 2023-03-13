import { NgModule } from '@angular/core';
import { RouterModule, Routes, UrlSegment } from '@angular/router';

import {
    AuthorizedGuard,
    ErrorsGuard,
    PermissionsGuard,
    SwitchServiceGuard,
    UnauthorizedGuard,
    WelcomeGuard,
    PageUrls
} from '@soldr/core';
import { LazyLoadTranslationsGuard } from '@soldr/i18n';

import { AuthorizedAreaComponent, UnauthorizedAreaComponent } from './routes';

const features = Object.keys(PageUrls).map((page) => page.toLowerCase());

function featureUrls(url: UrlSegment[]): { consumed: UrlSegment[] } | null {
    return features.includes(url[0].path) ? { consumed: url } : null;
}

const routes: Routes = [
    {
        path: '',
        pathMatch: 'full',
        canActivate: [LazyLoadTranslationsGuard, WelcomeGuard],
        children: []
    },
    {
        path: '',
        component: AuthorizedAreaComponent,
        canActivateChild: [AuthorizedGuard],
        children: [
            {
                path: 'services/:service_hash',
                canActivate: [SwitchServiceGuard],
                children: [
                    {
                        path: '',
                        canActivate: [LazyLoadTranslationsGuard],
                        canActivateChild: [PermissionsGuard],
                        data: { scope: ['agents', 'modules'] },
                        loadChildren: () => import('@soldr/features/agents').then((m) => m.FeaturesAgentsModule)
                    },
                    {
                        path: '',
                        canActivate: [LazyLoadTranslationsGuard],
                        canActivateChild: [PermissionsGuard],
                        data: { scope: ['groups', 'modules'] },
                        loadChildren: () => import('@soldr/features/groups').then((m) => m.FeaturesGroupsModule)
                    },
                    {
                        path: '',
                        canActivate: [LazyLoadTranslationsGuard],
                        canActivateChild: [PermissionsGuard],
                        data: { scope: ['policies', 'modules'] },
                        loadChildren: () => import('@soldr/features/policies').then((m) => m.FeaturesPoliciesModule)
                    },
                    {
                        path: '',
                        canActivate: [LazyLoadTranslationsGuard],
                        canActivateChild: [PermissionsGuard],
                        data: { scope: ['modules'] },
                        loadChildren: () => import('@soldr/features/modules').then((m) => m.FeaturesModulesModule)
                    },
                    {
                        path: '',
                        canActivate: [LazyLoadTranslationsGuard],
                        canActivateChild: [PermissionsGuard],
                        data: { scope: ['distributions'] },
                        loadChildren: () =>
                            import('@soldr/features/distributions').then((m) => m.FeaturesDistributionsModule)
                    }
                ]
            },
            {
                matcher: featureUrls,
                canActivate: [SwitchServiceGuard],
                children: []
            },
            {
                path: '',
                canActivate: [LazyLoadTranslationsGuard],
                canActivateChild: [ErrorsGuard],
                data: { scope: ['errors'] },
                loadChildren: () => import('@soldr/features/errors').then((m) => m.FeaturesErrorsModule)
            },
            {
                path: '',
                canActivate: [LazyLoadTranslationsGuard],
                canActivateChild: [],
                data: { scope: ['password'] },
                loadChildren: () => import('@soldr/features/password').then((m) => m.FeaturesPasswordModule)
            }
        ]
    },
    {
        path: '',
        component: UnauthorizedAreaComponent,
        canActivateChild: [UnauthorizedGuard],
        children: [
            {
                path: '',
                canActivate: [LazyLoadTranslationsGuard],
                data: { scope: ['login'] },
                loadChildren: () => import('@soldr/features/login').then((m) => m.FeaturesLoginModule)
            }
        ]
    },
    {
        path: '**',
        canActivate: [LazyLoadTranslationsGuard, WelcomeGuard],
        children: []
    }
];

@NgModule({
    imports: [
        RouterModule.forRoot(routes, {
            initialNavigation: 'enabledBlocking',
            useHash: false
        })
    ],
    exports: [RouterModule]
})
export class AppRoutingModule {}
