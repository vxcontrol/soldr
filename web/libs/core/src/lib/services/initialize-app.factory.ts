import { catchError, filter, map, of, take, tap } from 'rxjs';

import { PublicInfo, Themes } from '@soldr/api';
import { SharedFacade } from '@soldr/store/shared';

import { ThemeService } from './theme.service';

export const initializeApp = (sharedFacade: SharedFacade, themeService: ThemeService) => () =>
    sharedFacade.selectInfo().pipe(
        filter(Boolean),
        take(1),
        tap((info: PublicInfo) => themeService.setTheme(Themes.Dark)),
        map(() => true),
        catchError(() => of(false))
    );
