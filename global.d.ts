declare module 'eventsource' {
  class EventSource {
    constructor(url: string, eventSourceInitDict?: EventSourceInit);
    onmessage: ((this: EventSource, ev: MessageEvent) => any) | null;
    onerror: ((this: EventSource, ev: Event) => any) | null;
    close(): void;
  }

  interface EventSourceInit {
    withCredentials?: boolean;
  }
}
