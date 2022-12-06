import { InjectionToken } from '@angular/core';

import { MosaicTokens } from '../types';

import { ThemeService } from './theme.service';

export const THEME_TOKENS = new InjectionToken<MosaicTokens>('Mosaic theme tokens');
export const PROXY_THEME_SERVICE = (themeService: ThemeService) =>
    new Proxy({}, { get: (_, value: string) => (themeService.tokens as any)[value] });
