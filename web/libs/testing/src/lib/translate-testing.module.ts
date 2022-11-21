import { Injectable, NgModule, Pipe, PipeTransform } from '@angular/core';
import { TranslocoService, TranslocoModule, TranslocoPipe } from '@ngneat/transloco';
import { Observable, of } from 'rxjs';

@Pipe({
    name: 'transloco'
})
export class TranslocoPipeMock implements PipeTransform {
    transform(query: string): string {
        return query;
    }
}

@Injectable()
export class TranslocoServiceMock {
    selectTranslate<T>(key: T): Observable<T> {
        return of(key);
    }
}

@NgModule({
    declarations: [TranslocoPipeMock],
    exports: [TranslocoPipeMock, TranslocoModule],
    providers: [
        { provide: TranslocoService, useClass: TranslocoServiceMock },
        { provide: TranslocoPipe, useClass: TranslocoPipeMock }
    ]
})
export class TranslateTestingModule {}
