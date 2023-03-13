import { HttpClient } from '@angular/common/http';
import { Injectable } from '@angular/core';

import { BASE_URL } from '../base-urls';
import { ModelsPassword } from '../dto';

@Injectable({
    providedIn: 'root'
})
export class UserService {
    private readonly baseUrl = `${BASE_URL}/user/`;

    constructor(private http: HttpClient) {}

    changePassword(data: ModelsPassword) {
        return this.http.put(`${this.baseUrl}password`, data);
    }
}
