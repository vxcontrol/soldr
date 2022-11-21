import { HttpStatusCode } from '@angular/common/http';
import { Component, OnInit } from '@angular/core';
import { TranslocoService } from '@ngneat/transloco';
import { combineLatest, first } from 'rxjs';

import { PageTitleService } from '@soldr/shared';

@Component({
    templateUrl: './forbidden-page.component.html',
    styleUrls: ['./forbidden-page.component.scss']
})
export class ForbiddenPageComponent implements OnInit {
    forbidden = HttpStatusCode.Forbidden;

    constructor(private pageTitleService: PageTitleService, private transloco: TranslocoService) {}

    ngOnInit() {
        this.defineTitle();
    }

    private defineTitle() {
        combineLatest([
            this.transloco.selectTranslate('Errors.Forbidden.PageTitle.NoAccess', {}, 'errors'),
            this.transloco.selectTranslate('Shared.Pseudo.PageTitle.ApplicationName', {}, 'shared')
        ])
            .pipe(first())
            .subscribe((segments) => this.pageTitleService.setTitle(segments));
    }
}
