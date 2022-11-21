import { Injectable } from '@angular/core';
import { Title } from '@angular/platform-browser';

@Injectable({ providedIn: 'root' })
export class PageTitleService {
    private delimiter = ' \u00B7 ';

    constructor(private titleService: Title) {}

    setTitle(segments: string | string[] = []): void {
        const normalizedSegments = Array.isArray(segments) ? segments : Array.of(segments);

        this.titleService.setTitle(normalizedSegments.join(this.delimiter));
    }
}
