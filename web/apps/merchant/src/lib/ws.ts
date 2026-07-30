/**
 * عميل البث الحي: اتصال WebSocket بتوكن الوصول، إعادة اتصال تلقائية
 * بتراجع أسّي، وإرجاع حالة الاتصال للواجهة.
 */

import { useEffect, useRef, useState } from "react";
import { tokenStore } from "./api";

const API_URL = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";

export function useLiveEvents(onEvent: (event: { type: string; [k: string]: unknown }) => void) {
  const [connected, setConnected] = useState(false);
  const handler = useRef(onEvent);
  handler.current = onEvent;

  useEffect(() => {
    let ws: WebSocket | null = null;
    let closed = false;
    let backoff = 1000;
    let timer: ReturnType<typeof setTimeout>;

    function connect() {
      const token = tokenStore.access;
      if (!token || closed) return;
      const url = API_URL.replace(/^http/, "ws") + `/api/v1/ws?token=${encodeURIComponent(token)}`;
      ws = new WebSocket(url);

      ws.onopen = () => {
        setConnected(true);
        backoff = 1000;
      };
      ws.onmessage = (e) => {
        try {
          handler.current(JSON.parse(e.data));
        } catch {
          // رسالة غير صالحة — تجاهل
        }
      };
      ws.onclose = () => {
        setConnected(false);
        if (closed) return;
        timer = setTimeout(connect, backoff);
        backoff = Math.min(backoff * 2, 15000);
      };
      ws.onerror = () => ws?.close();
    }

    connect();
    return () => {
      closed = true;
      clearTimeout(timer);
      ws?.close();
    };
  }, []);

  return connected;
}
