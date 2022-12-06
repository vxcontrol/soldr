import { Injectable } from '@angular/core';

import { Themes } from '@soldr/api';
import { DarkThemeTokens, LightThemeTokens } from '@soldr/styles';

import { MonacoService } from './monaco.service';

@Injectable({
    providedIn: 'root'
})
export class ThemeService {
    theme: Themes;

    constructor(private monacoService: MonacoService) {}

    get tokens() {
        return this.theme === Themes.Light ? LightThemeTokens : DarkThemeTokens;
    }

    setTheme(theme: Themes) {
        this.theme = theme;

        if (theme === Themes.Dark) {
            document.body.classList.remove('light-theme');
            document.body.classList.add('dark-theme');
        } else {
            document.body.classList.remove('dark-theme');
            document.body.classList.add('light-theme');
        }

        this.monacoService.defineTheme(theme, this.tokens);
    }
}
