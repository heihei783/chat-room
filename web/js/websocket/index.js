/**
 * WebSocket 客户端封装：连接管理、心跳保活、断线重连、消息分发。
 * 服务端协议（见后端 pkg/websocket/message.go）：
 *   - 连接：GET /ws?token=<JWT>（浏览器原生 WS 无法自定义 header，token 走 query）
 *   - 客户端可发：{type:'ping'}、{type:'join', room_id}、{type:'leave', room_id}
 *   - 服务端推送：{type:'message'|'presence'|'notification'|'pong'}
 * 职责边界：本模块只负责连接与转发，消息的业务归置委托给对应 service。
 */
import { ENV } from '../config/env.js';
import { getToken } from '../api/http.js';
import { setState, getState } from '../store/index.js';
import { messageService } from '../services/messageService.js';
import { presenceService } from '../services/presenceService.js';
import { notificationService } from '../services/notificationService.js';

/** 心跳间隔：每 25 秒发送 ping，服务端回应 pong 即视为存活 */
const HEARTBEAT_INTERVAL = 25000;
/** 断线重连：指数退避 1s→2s→4s…上限 30s */
const RECONNECT_BASE = 1000;
const RECONNECT_MAX = 30000;

class WebSocketClient {
  constructor() {
    this.socket = null;
    this.heartbeatTimer = null;
    this.reconnectTimer = null;
    this.reconnectDelay = RECONNECT_BASE;
    this.manuallyClosed = false;
  }

  /** 发起连接：需已登录（有 token） */
  connect() {
    const token = getToken();
    if (!token) return;
    this.manuallyClosed = false;
    this.open(token);
  }

  /** 打开底层连接 */
  open(token) {
    try {
      this.socket = new WebSocket(`${ENV.WS_BASE}/ws?token=${encodeURIComponent(token)}`);
    } catch {
      this.scheduleReconnect();
      return;
    }

    this.socket.onopen = () => {
      this.reconnectDelay = RECONNECT_BASE;
      setState({ wsConnected: true });
      this.startHeartbeat();
      // 重连后恢复当前群聊会话的订阅
      const { activeRoom } = getState();
      if (activeRoom) this.joinRoom(activeRoom.id);
    };

    this.socket.onmessage = (event) => this.dispatch(event.data);

    this.socket.onclose = () => {
      setState({ wsConnected: false });
      this.stopHeartbeat();
      this.scheduleReconnect();
    };

    this.socket.onerror = () => {
      this.socket && this.socket.close();
    };
  }

  /** 按 type 分发服务端推送消息 */
  dispatch(raw) {
    let data;
    try {
      data = JSON.parse(raw);
    } catch {
      return; // 非 JSON 帧直接忽略
    }
    switch (data.type) {
      case 'message': {
        // 协议帧 type 固定为 message，消息本身类型在 msg_type（text/image/file），归一化后交给业务层
        const payload = data.data || data.message || data;
        messageService.handleRealtimeMessage({ ...payload, type: payload.msg_type || payload.type });
        break;
      }
      case 'presence':
        presenceService.handlePresenceBroadcast(data.data || data);
        break;
      case 'notification':
        notificationService.handleRealtimeNotification(data.data || data.notification || data);
        break;
      case 'pong':
        break; // 心跳回应，无需处理
      default:
        break;
    }
  }

  /** 发送原始 JSON 帧（连接未就绪时静默丢弃） */
  send(data) {
    if (this.socket && this.socket.readyState === WebSocket.OPEN) {
      this.socket.send(JSON.stringify(data));
    }
  }

  /** 订阅群聊房间：加入后服务端向该房间广播群聊消息 */
  joinRoom(roomId) {
    this.send({ type: 'join', room_id: roomId });
  }

  /** 取消订阅群聊房间 */
  leaveRoom(roomId) {
    this.send({ type: 'leave', room_id: roomId });
  }

  /** 心跳：定时发送 ping */
  startHeartbeat() {
    this.stopHeartbeat();
    this.heartbeatTimer = setInterval(() => this.send({ type: 'ping' }), HEARTBEAT_INTERVAL);
  }

  stopHeartbeat() {
    if (this.heartbeatTimer) {
      clearInterval(this.heartbeatTimer);
      this.heartbeatTimer = null;
    }
  }

  /** 指数退避重连（手动关闭时不重连） */
  scheduleReconnect() {
    if (this.manuallyClosed) return;
    if (this.reconnectTimer) return;
    this.reconnectTimer = setTimeout(() => {
      this.reconnectTimer = null;
      const token = getToken();
      if (token) this.open(token);
    }, this.reconnectDelay);
    this.reconnectDelay = Math.min(this.reconnectDelay * 2, RECONNECT_MAX);
  }

  /** 主动断开（登出时调用）：停止心跳与重连，关闭连接 */
  close() {
    this.manuallyClosed = true;
    this.stopHeartbeat();
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer);
      this.reconnectTimer = null;
    }
    if (this.socket) {
      this.socket.onclose = null;
      this.socket.close();
      this.socket = null;
    }
    setState({ wsConnected: false });
  }
}

/** 单例导出：全局只有一个 WS 连接 */
export const wsClient = new WebSocketClient();

/** 便捷导出：登出时调用 */
export function closeSocket() {
  wsClient.close();
}
