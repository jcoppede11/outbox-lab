import { ChangeDetectionStrategy, Component, computed, input } from '@angular/core';

import { BrokerState, OutboxRow, Payment } from './models';

/**
 * Invariante siempre visible: "todo pago registrado acaba notificado".
 * Con Outbox, `notificados` alcanza a `registrados` (los pendientes están en
 * tránsito, nunca perdidos). Un desajuste permanente delataría cobros huérfanos.
 */
@Component({
  selector: 'app-invariant-counter',
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <section
      class="rounded-xl border p-4 shadow-sm transition-colors"
      [class]="
        balanced()
          ? 'border-emerald-300 bg-emerald-50'
          : 'border-amber-300 bg-amber-50'
      "
    >
      <div class="flex flex-wrap items-center justify-between gap-4">
        <div>
          <p class="text-xs font-medium uppercase tracking-wide text-slate-500">Invariante</p>
          <p class="text-lg font-semibold text-slate-800">
            Todo pago registrado acaba notificado
          </p>
        </div>
        <div class="flex items-center gap-6">
          <div class="text-center">
            <p class="text-2xl font-bold text-slate-900">{{ notified() }} / {{ registered() }}</p>
            <p class="text-xs text-slate-500">notificados / registrados</p>
          </div>
          <div
            class="rounded-full px-3 py-1 text-sm font-semibold"
            [class]="
              balanced()
                ? 'bg-emerald-200 text-emerald-800'
                : 'bg-amber-200 text-amber-800'
            "
          >
            @if (balanced()) {
              ✓ en equilibrio
            } @else {
              ⏳ {{ inFlight() }} en tránsito
            }
          </div>
        </div>
      </div>
    </section>
  `,
})
export class InvariantCounter {
  readonly payments = input.required<Payment[]>();
  readonly outbox = input.required<OutboxRow[]>();
  readonly broker = input.required<BrokerState | null>();

  protected readonly registered = computed(() => this.payments().length);

  /** Pagos distintos que el broker realmente recibió (idempotente ante duplicados). */
  protected readonly notified = computed(
    () => new Set((this.broker()?.received ?? []).map((e) => e.payment_id)).size,
  );

  /** Eventos aún no enviados (pendientes o fallidos): en tránsito, no perdidos. */
  protected readonly inFlight = computed(
    () => this.outbox().filter((r) => r.status !== 'sent').length,
  );

  protected readonly balanced = computed(
    () => this.inFlight() === 0 && this.notified() >= this.registered(),
  );
}
