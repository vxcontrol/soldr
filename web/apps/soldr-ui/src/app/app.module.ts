import { APP_INITIALIZER, NgModule } from '@angular/core';
import { BrowserModule } from '@angular/platform-browser';
import { BrowserAnimationsModule } from '@angular/platform-browser/animations';
import { ApiModule } from '@soldr/api';
import { CoreModule } from '@soldr/core';
import { environment } from '@soldr/environments';
import { I18nModule } from '@soldr/i18n';
import { SharedModule } from '@soldr/shared';
import { StoreAgentsModule } from '@soldr/store/agents';
import { StoreGroupsModule } from '@soldr/store/groups';
import { StoreModulesModule } from '@soldr/store/modules';
import { StoreModulesInstancesModule } from '@soldr/store/modules-instances';
import { StorePoliciesModule } from '@soldr/store/policies';
import { StoreSharedModule } from '@soldr/store/shared';
import { StoreTagsModule } from '@soldr/store/tags';

import { AppRoutingModule } from './app-routing.module';
import { AppComponent } from './app.component';
import { NavbarComponent } from './components';
import { AuthorizedAreaComponent, UnauthorizedAreaComponent } from './routes';
import { LoadingBarHttpClientModule } from '@ngx-loading-bar/http-client';
import { LoadingBarRouterModule } from '@ngx-loading-bar/router';
import { LoadingBarModule } from '@ngx-loading-bar/core';
import { EffectsModule } from '@ngrx/effects';
import { StoreDevtoolsModule } from '@ngrx/store-devtools';
import { StoreModule } from '@ngrx/store';
import { initializeTheme } from './utils';

@NgModule({
    declarations: [AppComponent, AuthorizedAreaComponent, NavbarComponent, UnauthorizedAreaComponent],
    imports: [
        BrowserAnimationsModule,
        BrowserModule,

        LoadingBarHttpClientModule,
        LoadingBarRouterModule,
        LoadingBarModule,

        ApiModule,
        CoreModule,
        I18nModule,
        SharedModule,

        StoreModule.forRoot(
            {},
            {
                runtimeChecks: {
                    strictStateImmutability: true,
                    strictActionImmutability: true
                }
            }
        ),
        EffectsModule.forRoot([]),
        StoreDevtoolsModule.instrument({
            maxAge: 25,
            logOnly: environment.production,
            autoPause: true
        }),
        StoreSharedModule,
        StoreAgentsModule,
        StoreGroupsModule,
        StoreModulesModule,
        StoreTagsModule,
        StoreModulesInstancesModule,
        StorePoliciesModule,

        AppRoutingModule
    ],
    providers: [
        {
            provide: APP_INITIALIZER,
            useFactory: () => initializeTheme,
            multi: true
        }
    ],
    bootstrap: [AppComponent]
})
export class AppModule {}
