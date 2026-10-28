import type { ChatThread } from '@/core/api/types';
import { pageTestProviders } from '@/testing/providers';
import { ChatThreadView } from '@/ui/chat-thread';
import { HttpTestingController } from '@angular/common/http/testing';
import { TestBed } from '@angular/core/testing';

const THREAD: ChatThread = {
  id: 'chat1',
  partnerId: 'partner',
  userId: 'me',
  role: 'client',
  peer: { id: 'partner', firstName: 'Айгуль', username: null, avatarUrl: null },
  lastMessage: { id: 'm2', senderId: 'me', body: 'Спасибо', createdAt: '2026-10-10T07:00:00Z' },
  unread: 0,
  peerReadAt: '2026-10-10T07:05:00Z',
  messages: [
    { id: 'm1', senderId: 'partner', body: 'Сбор в 6:00', createdAt: '2026-10-10T06:00:00Z' },
    { id: 'm2', senderId: 'me', body: 'Спасибо', createdAt: '2026-10-10T07:00:00Z' },
  ],
};

describe('ChatThreadView', () => {
  it('shows the messages, the visitor’s on the right, and sends a new one', async () => {
    TestBed.configureTestingModule({ providers: pageTestProviders() });
    const fixture = TestBed.createComponent(ChatThreadView);
    fixture.componentRef.setInput('chatId', 'chat1');
    fixture.componentRef.setInput('me', 'me');
    fixture.detectChanges();
    const http = TestBed.inject(HttpTestingController);
    http.expectOne('/chats/chat1/messages').flush(THREAD);
    await fixture.whenStable();

    const element: HTMLElement = fixture.nativeElement;
    expect(element.querySelector('header')?.textContent).toContain('Айгуль');
    const bubbles = Array.from(
      element.querySelectorAll('.items-end p:first-child, .items-start p:first-child'),
    );
    expect(bubbles.map((b) => b.textContent?.trim())).toEqual(['Сбор в 6:00', 'Спасибо']);
    expect(element.querySelector('.items-end')?.textContent).toContain('chat.read');

    const box = element.querySelector('textarea') as HTMLTextAreaElement;
    box.value = '  До встречи  ';
    box.dispatchEvent(new Event('input'));
    element.querySelector('form')?.dispatchEvent(new Event('submit'));
    const send = http.expectOne({ method: 'POST', url: '/chats/chat1/messages' });
    expect(send.request.body).toEqual({ body: 'До встречи' });
    send.flush({ id: 'm3', senderId: 'me', body: 'До встречи', createdAt: '2026-10-10T08:00:00Z' });
    await new Promise((resolve) => setTimeout(resolve));
    fixture.detectChanges();
    expect(element.textContent).toContain('До встречи');
    expect(box.value).toBe('');
  });
});
