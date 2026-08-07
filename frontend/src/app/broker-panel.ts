import { ChangeDetectionStrategy, Component, input } from '@angular/core';
import { DecimalPipe } from '@angular/common';

import { BrokerState } from './models';

/** Panel del broker: estado (arriba/caído) y eventos realmente recibidos. */
@Component({
  selector: 'app-broker-panel',
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [DecimalPipe],
  template: `
    <section class="flex h-full flex-col rounded-xl border border-slate-200 bg-white shadow-sm">
      <header class="flex items-center justify-between border-b border-slate-100 px-4 py-3">
        <h2 class="font-semibold text-slate-800">📨 Broker</h2>
        @if (broker(); as b) {
          <span
            class="rounded-full px-2 py-0.5 text-xs font-medium"
            [class]="b.down ? 'bg-rose-100 text-rose-700' : 'bg-emerald-100 text-emerald-700'"
          >
            {{ b.down ? 'caído' : 'activo' }}
          </span>
        }
      </header>
      <div class="flex-1 overflow-auto">
        @if (broker(); as b) {
          <p class="border-b border-slate-100 px-4 py-2 text-sm text-slate-600">
            Recibidos: <span class="font-semibold text-slate-900">{{ b.received_count }}</span>
          </p>
          @if (b.received.length === 0) {
            <p class="p-4 text-sm text-slate-400">Nada recibido todavía.</p>
          } @else {
            <ul class="divide-y divide-slate-100">
              @for (ev of b.received; track $index) {
                <li class="flex items-center justify-between px-4 py-2 text-sm">
                  <span class="truncate font-mono text-xs text-slate-500">
                    {{ ev.payment_id.slice(0, 8) }}
                  </span>
                  <span class="font-medium text-slate-800">
                    {{ ev.amount | number: '1.2-2' }} {{ ev.currency }}
                  </span>
                </li>
              }
            </ul>
          }
        } @else {
          <p class="p-4 text-sm text-slate-400">Conectando…</p>
        }
      </div>
    </section>
  `,
})
export class BrokerPanel {
  readonly broker = input.required<BrokerState | null>();
}
