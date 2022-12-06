import { Component, Inject, OnDestroy } from '@angular/core';
import { TranslocoService } from '@ngneat/transloco';
import { Subscription } from 'rxjs';

import { MosaicTokens, THEME_TOKENS } from '@soldr/core';
import { SharedFacade } from '@soldr/store/shared';
import { PublicInfo } from '@soldr/api';

@Component({
    selector: 'soldr-root',
    templateUrl: './app.component.html',
    styleUrls: ['./app.component.scss'],
    providers: []
})
export class AppComponent implements OnDestroy {
    private subscription: Subscription = new Subscription();

    constructor(private sharedFacade: SharedFacade, private transloco: TranslocoService,
        @Inject(THEME_TOKENS) public tokens: MosaicTokens
    ) {
        const localesChangesSubscription = this.transloco.langChanges$.subscribe((locale) => {
            localStorage.setItem('locale', locale);
        });
        this.subscription.add(localesChangesSubscription);
    }

    ngOnDestroy() {
        this.subscription.unsubscribe();
    }
}
