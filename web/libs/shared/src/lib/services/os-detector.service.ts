import { Injectable } from '@angular/core';

@Injectable({
    providedIn: 'root'
})
export class OsDetectorService {
    isWindows = navigator.userAgent.indexOf('Win') !== -1;
    isMacOS = navigator.userAgent.indexOf('Mac') !== -1;
    isLinux = navigator.userAgent.indexOf('Linux') !== -1;
    isNotMacOS = navigator.userAgent.indexOf('Mac') === -1;
}
