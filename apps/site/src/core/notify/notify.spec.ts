import { Notify } from '@/core/notify/notify';
import { PLATFORM_ID } from '@angular/core';
import { TestBed } from '@angular/core/testing';

function notify(platform: 'browser' | 'server' = 'browser'): Notify {
  TestBed.configureTestingModule({ providers: [{ provide: PLATFORM_ID, useValue: platform }] });
  return TestBed.inject(Notify);
}

describe('Notify', () => {
  it('gives each kind its lifetime; errors stay until closed', () => {
    const service = notify();
    service.success({ title: 'a' });
    service.error({ title: 'b' });
    const [error, success] = service.toasts();
    expect(success?.duration).toBe(4000);
    expect(error?.duration).toBeNull();
  });

  it('counts a repeated message instead of stacking it', () => {
    const service = notify();
    const first = service.info({ title: 'saved' });
    const again = service.info({ title: 'saved' });
    expect(service.toasts()).toHaveLength(1);
    expect(service.toasts()[0]?.count).toBe(2);
    // A new id restarts the toast's timer.
    expect(again).not.toBe(first);
  });

  it('keeps at most three, newest first', () => {
    const service = notify();
    for (const title of ['1', '2', '3', '4']) service.success({ title });
    expect(service.toasts().map((toast) => toast.title)).toEqual(['4', '3', '2']);
  });

  it('dismisses by id', () => {
    const service = notify();
    const id = service.warning({ title: 'x' });
    service.dismiss(id);
    expect(service.toasts()).toEqual([]);
  });

  it('queues nothing while rendering on the server', () => {
    const service = notify('server');
    expect(service.success({ title: 'x' })).toBe(0);
    expect(service.toasts()).toEqual([]);
  });
});
