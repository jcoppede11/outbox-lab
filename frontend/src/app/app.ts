import { ChangeDetectionStrategy, Component, inject } from '@angular/core';

import { PaymentService } from './payment.service';
import { Controls } from './controls';
import { InvariantCounter } from './invariant-counter';
import { PaymentsPanel } from './payments-panel';
import { OutboxPanel } from './outbox-panel';
import { BrokerPanel } from './broker-panel';

@Component({
  selector: 'app-root',
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [Controls, InvariantCounter, PaymentsPanel, OutboxPanel, BrokerPanel],
  templateUrl: './app.html',
})
export class App {
  protected readonly payments = inject(PaymentService);
}
