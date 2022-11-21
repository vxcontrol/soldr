import { HttpClient } from '@angular/common/http';
import { Injectable } from '@angular/core';
import { Translation, TranslocoLoader } from '@ngneat/transloco';
import { TranslocoLoaderData } from '@ngneat/transloco/lib/transloco.loader';
import { Observable, of } from 'rxjs';
import { catchError } from 'rxjs/operators';

@Injectable({ providedIn: 'root' })
export class TranslocoHttpLoaderService implements TranslocoLoader {
    constructor(private httpClient: HttpClient) {}

    getTranslation(path: string, data?: TranslocoLoaderData): Observable<Translation> {
        if (!data?.scope) {
            // NOTE: Все файлы i18n должны относиться к какому-то конкретному TRANSLOCO_SCOPE, иначе в метод прилетает
            // path = 'ru-RU' и data = {scope: null} => мы не знаем к какому файлу обращаться с запросом;
            // возвращаем пустой объект, т.к. это самый лёгкий способ обработать ошибку через MissingKeyHandler
            // return throwError() тут ни кем не обработается...
            return of({});
        }

        const pathParts = path.split('/');
        const lang = pathParts[pathParts.length - 1];

        return this.httpClient.get<Translation>(`/assets/i18n/${lang}/${data.scope}.json`).pipe(
            // NOTE: В случае, когда файл не найден, хотя бы возвращаем пустой объект и не ломаем приложение
            catchError(() => of({}))
        );
    }
}
