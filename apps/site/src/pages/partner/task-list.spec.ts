import type { CrmTask } from '@/core/api/types';
import { todayInBishkek } from '@/core/lib/format';
import { TaskList } from '@/pages/partner/task-list';
import { pageTestProviders } from '@/testing/providers';
import { HttpTestingController } from '@angular/common/http/testing';
import { TestBed } from '@angular/core/testing';

const task = (fields: Partial<CrmTask>): CrmTask => ({
  id: 't1',
  title: 'Перезвонить',
  dueOn: '2026-01-01',
  done: false,
  client: { id: 'c1', userId: null, name: 'Айбек', account: null },
  tour: null,
  doneAt: null,
  ...fields,
});

describe('TaskList', () => {
  it('shows overdue tasks in red, adds and completes them', async () => {
    TestBed.configureTestingModule({ providers: pageTestProviders() });
    const fixture = TestBed.createComponent(TaskList);
    fixture.detectChanges();
    const http = TestBed.inject(HttpTestingController);
    http.expectOne('/crm/tasks').flush([task({})]);
    await fixture.whenStable();
    const element: HTMLElement = fixture.nativeElement;
    expect(element.textContent).toContain('Перезвонить');
    expect(element.textContent).toContain('Айбек');
    expect(element.querySelector('.text-danger')).not.toBeNull();

    const input = element.querySelector('#task-title') as HTMLInputElement;
    input.value = ' Купить газ ';
    input.dispatchEvent(new Event('input'));
    element.querySelector('form')?.dispatchEvent(new Event('submit'));
    const add = http.expectOne({ method: 'POST', url: '/crm/tasks' });
    expect(add.request.body).toEqual({ title: 'Купить газ', dueOn: todayInBishkek() });
    add.flush([]);
    await new Promise((resolve) => setTimeout(resolve));
    http
      .expectOne('/crm/tasks')
      .flush([task({}), task({ id: 't2', title: 'Купить газ', dueOn: todayInBishkek() })]);
    await fixture.whenStable();
    expect(element.textContent).toContain('crm.tasks.today');

    (element.querySelector('input[type="checkbox"]') as HTMLInputElement).dispatchEvent(
      new Event('change'),
    );
    const done = http.expectOne({ method: 'PATCH', url: '/crm/tasks/t1' });
    expect(done.request.body).toEqual({ done: true });
  });
});
