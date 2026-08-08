import { Injectable, computed, inject, signal } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { toSignal } from '@angular/core/rxjs-interop';
import { Subject, catchError, merge, of, switchMap, tap, timer } from 'rxjs';

import { AppState, CreatePaymentRequest } from './models';

/** Período del polling de /api/state, en milisegundos. */
const POLL_MS = 3000;

/**
 * Estado central de la demo. Hace polling de GET /api/state cada 3 segundos para
 * ver los eventos "viajar" de la outbox al broker en tiempo real, y expone
 * acciones (crear pago, modo caos) que refrescan el estado inmediatamente.
 *
 * El estado se publica como signals, así que en modo zoneless la UI se
 * actualiza sola en cada emisión.
 */
@Injectable({ providedIn: 'root' })
export class PaymentService {
  private readonly http = inject(HttpClient);

  /** Dispara un refresco inmediato además del tick periódico. */
  private readonly refresh = new Subject<void>();

  /** Último estado bueno, para no vaciar la UI ante un error transitorio. */
  private lastGood: AppState | null = null;
  private readonly connectionError = signal(false);

  private readonly state$ = merge(timer(0, POLL_MS), this.refresh).pipe(
    switchMap(() =>
      this.http.get<AppState>('/api/state').pipe(
        tap((s) => {
          this.lastGood = s;
          this.connectionError.set(false);
        }),
        catchError(() => {
          this.connectionError.set(true);
          return of(this.lastGood);
        }),
      ),
    ),
  );

  /** Estado actual (payments, outbox, broker). `null` hasta la primera respuesta. */
  readonly state = toSignal(this.state$, { initialValue: null });

  /** `true` cuando el backend no responde. */
  readonly offline = this.connectionError.asReadonly();

  /** Convenience signals derivadas. */
  readonly payments = computed(() => this.state()?.payments ?? []);
  readonly outbox = computed(() => this.state()?.outbox ?? []);
  readonly broker = computed(() => this.state()?.broker ?? null);

  /** Registra un pago (y su evento outbox) y refresca al confirmar. */
  createPayment(req: CreatePaymentRequest): void {
    this.http.post('/api/payments', req).subscribe({
      next: () => this.refresh.next(),
      error: () => this.refresh.next(),
    });
  }

  /** Tira (`true`) o levanta (`false`) el broker. */
  setBrokerDown(down: boolean): void {
    this.http.post('/api/chaos', { down }).subscribe({
      next: () => this.refresh.next(),
      error: () => this.refresh.next(),
    });
  }
}
