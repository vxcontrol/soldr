import { Component, OnDestroy } from '@angular/core';
import { TranslocoService } from '@ngneat/transloco';
import { Observable, Subscription } from 'rxjs';

import { SharedFacade } from '@soldr/store/shared';
import { PublicInfo } from '@soldr/api';

@Component({
    selector: 'soldr-root',
    templateUrl: './app.component.html',
    styleUrls: ['./app.component.scss'],
    providers: []
})
export class AppComponent implements OnDestroy {
    info: PublicInfo;
    loadingBarColor: string;

    private subscription: Subscription = new Subscription();

    constructor(private sharedFacade: SharedFacade, private transloco: TranslocoService) {
        const localesChangesSubscription = this.transloco.langChanges$.subscribe((locale) => {
            localStorage.setItem('locale', locale);
        });
        this.subscription.add(localesChangesSubscription);

        this.loadingBarColor = getComputedStyle(document.documentElement).getPropertyValue('--loading-bar-color');
    }

    ngOnDestroy() {
        this.subscription.unsubscribe();
    }
}
