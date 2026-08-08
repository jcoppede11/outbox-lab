import { ChangeDetectionStrategy, Component, computed, inject } from '@angular/core';
import { FormBuilder, ReactiveFormsModule, Validators } from '@angular/forms';

import { PaymentService } from './payment.service';

/** Controles interactivos: crear un pago y tirar/levantar el broker (modo caos). */
@Component({
  selector: 'app-controls',
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [ReactiveFormsModule],
  template: `
    <section>
      <div class="flex flex-wrap items-end gap-3">
        <form class="contents" [formGroup]="form" (ngSubmit)="submit()">
          <div class="flex flex-col gap-1">
            <label class="mb-1 text-xs font-medium text-slate-500" for="customer">Cliente</label>
            <input
              id="customer"
              type="text"
              formControlName="customer"
              placeholder="alice"
              class="rounded-md border border-slate-300 px-3 py-1.5 text-sm focus:border-slate-500 focus:outline-none"
            />
          </div>
          <div class="flex flex-col gap-1">
            <label class="mb-1 text-xs font-medium text-slate-500" for="amount">Monto</label>
            <input
              id="amount"
              type="number"
              step="0.01"
              min="0.01"
              formControlName="amount"
              class="w-28 rounded-md border border-slate-300 px-3 py-1.5 text-sm focus:border-slate-500 focus:outline-none"
            />
          </div>
          <div class="flex flex-col gap-1">
            <label class="mb-1 text-xs font-medium text-slate-500" for="currency">Moneda</label>
            <select
              id="currency"
              formControlName="currency"
              class="rounded-md border border-slate-300 px-3 py-1.5 text-sm focus:border-slate-500 focus:outline-none"
            >
              <option value="USD">USD</option>
              <option value="EUR">EUR</option>
              <option value="ARS">ARS</option>
            </select>
          </div>
          <button
            type="submit"
            [disabled]="form.invalid"
            class="rounded-md bg-slate-500 px-4 py-1.5 text-sm font-semibold text-white hover:bg-slate-700 disabled:cursor-not-allowed disabled:opacity-50"
          >
            Crear pago
          </button>
        </form>

        <span class="mx-2 h-8 w-px bg-slate-200" aria-hidden="true"></span>

        <button
          type="button"
          (click)="toggleChaos()"
          [attr.aria-pressed]="brokerDown()"
          class="rounded-md px-4 py-1.5 text-sm font-semibold text-white"
          [class]="brokerDown() ? 'bg-emerald-600 hover:bg-emerald-700' : 'bg-rose-600 hover:bg-rose-700'"
        >
          {{ brokerDown() ? 'Levantar broker' : 'Tirar broker' }}
        </button>
      </div>
    </section>
  `,
})
export class Controls {
  private readonly fb = inject(FormBuilder);
  private readonly payments = inject(PaymentService);

  protected readonly brokerDown = computed(() => this.payments.broker()?.down ?? false);

  protected readonly form = this.fb.nonNullable.group({
    customer: ['Alice', Validators.required],
    amount: [10, [Validators.required, Validators.min(0.01)]],
    currency: ['USD', Validators.required],
  });

  protected submit(): void {
    if (this.form.invalid) {
      return;
    }
    this.payments.createPayment(this.form.getRawValue());
  }

  protected toggleChaos(): void {
    this.payments.setBrokerDown(!this.brokerDown());
  }
}
