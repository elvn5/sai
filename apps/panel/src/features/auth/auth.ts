import { HttpClient } from '@angular/common/http';
import { Injectable, inject } from '@angular/core';
import type { CanActivateFn } from '@angular/router';
import { Router } from '@angular/router';
import { catchError, firstValueFrom, map, of } from 'rxjs';

// The admin session: a password login that sets the admin_token cookie.
@Injectable({ providedIn: 'root' })
export class AdminAuth {
  private readonly http = inject(HttpClient);

  login(password: string): Promise<unknown> {
    return firstValueFrom(this.http.post('/auth/login', { password }));
  }

  logout(): Promise<unknown> {
    return firstValueFrom(this.http.post('/auth/logout', {}));
  }
}

// Lets signed-in admins through; everyone else goes to /login.
export const requireAdmin: CanActivateFn = () => {
  const router = inject(Router);
  return inject(HttpClient)
    .get('/auth/me')
    .pipe(
      map(() => true),
      catchError(() => of(router.createUrlTree(['/login']))),
    );
};
