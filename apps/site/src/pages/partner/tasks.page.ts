import { bindPageMeta } from '@/core/seo/seo.service';
import { TaskList } from '@/pages/partner/task-list';
import { ChangeDetectionStrategy, Component, inject } from '@angular/core';
import { TranslocoPipe, TranslocoService } from '@jsverse/transloco';

// /partner/tasks: all the partner's open tasks by due day.
@Component({
  selector: 'app-partner-tasks-page',
  imports: [TaskList, TranslocoPipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="flex flex-col gap-5">
      <div>
        <h1 class="text-2xl font-extrabold tracking-tight text-text">{{ 'cabinet.nav.tasks' | transloco }}</h1>
        <p class="text-sm text-text-muted">{{ 'crm.tasks.subtitle' | transloco }}</p>
      </div>
      <section class="ui-card p-4"><app-task-list /></section>
    </div>
  `,
})
export class PartnerTasksPage {
  private readonly transloco = inject(TranslocoService);

  constructor() {
    bindPageMeta(() => ({
      title: this.transloco.translate('cabinet.nav.tasks'),
      description: this.transloco.translate('crm.tasks.subtitle'),
      robots: 'noindex',
    }));
  }
}
