import { Injectable } from '@angular/core';
import * as monaco from 'monaco-editor/esm/vs/editor/editor.api';

import { Themes } from '@soldr/api';
import { capitalize } from '@soldr/shared';

import { MosaicTokens } from '../types';

@Injectable({
    providedIn: 'root'
})
export class MonacoService {
    constructor() {}

    defineTheme(theme: Themes, tokens: MosaicTokens) {
        const prefix = capitalize(theme) as Capitalize<Themes>;
        const colors = {
            text: tokens[`${prefix}ColorSchemeForegroundText`],
            textLessContrast: tokens[`${prefix}ColorSchemeForegroundTextLessContrast`],
            background: tokens[`${prefix}ColorSchemeBackgroundBackground`]
        };

        monaco.editor.defineTheme('soldrJsonTheme', {
            base: theme === Themes.Dark ? 'vs-dark' : 'vs',
            inherit: true,
            rules: [
                {
                    token: 'string.key.json',
                    foreground: colors.textLessContrast
                },
                {
                    token: 'string.value.json',
                    foreground: colors.text
                },
                { token: 'number', foreground: colors.text },
                { token: 'keyword.json', foreground: colors.text }
            ],
            colors: {
                'editor.foreground': colors.text,
                'editor.background': colors.background
            }
        });

        monaco.editor.defineTheme('soldrTheme', {
            base: theme === Themes.Dark ? 'vs-dark' : 'vs',
            inherit: true,
            rules: [{ token: '', background: colors.background }],
            colors: {
                'editor.background': colors.background
            }
        });

        monaco.editor.setTheme('soldrTheme');
    }
}
