"use client";

import { useEffect, useRef } from "react";

import { WS_URL } from "@/lib/api";
import type { RealtimeMessage } from "@/types/market";

interface Options {
  onMessage: (message: RealtimeMessage) => void;
  onStatusChange?: (connected: boolean) => void;
}

export function useRealtime({ onMessage, onStatusChange }: Options): void {
  const callbackRef = useRef(onMessage);
  const statusRef = useRef(onStatusChange);

  useEffect(() => {
    callbackRef.current = onMessage;
  }, [onMessage]);

  useEffect(() => {
    statusRef.current = onStatusChange;
  }, [onStatusChange]);

  useEffect(() => {
    let socket: WebSocket | null = null;
    let retryTimer: ReturnType<typeof setTimeout> | null = null;
    let stopped = false;
    let attempt = 0;

    const connect = () => {
      if (stopped) return;
      socket = new WebSocket(WS_URL);

      socket.addEventListener("open", () => {
        attempt = 0;
        statusRef.current?.(true);
      });

      socket.addEventListener("message", (event) => {
        try {
          const parsed = JSON.parse(event.data) as RealtimeMessage;
          callbackRef.current(parsed);
        } catch {
          // Ignore malformed events.
        }
      });

      socket.addEventListener("close", () => {
        statusRef.current?.(false);
        if (stopped) return;
        attempt += 1;
        const delay = Math.min(1000 * 2 ** attempt, 15000);
        retryTimer = setTimeout(connect, delay);
      });

      socket.addEventListener("error", () => {
        socket?.close();
      });
    };

    connect();

    return () => {
      stopped = true;
      statusRef.current?.(false);
      if (retryTimer) clearTimeout(retryTimer);
      socket?.close();
    };
  }, []);
}
