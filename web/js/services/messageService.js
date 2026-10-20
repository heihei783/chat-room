/**
 * 消息业务服务：发送/查询/已读/未读数，以及 WebSocket 实时消息的归置。
 * 页面发送消息、切换会话均通过本服务；实时消息也由本服务统一处理（唯一入口）。
 */
import { messageApi } from '../api/message.js';
import { getState, setState } from '../store/index.js';
import { notificationService } from './notificationService.js';
import { toast } from '../components/Toast.js';

export const messageService = {
  /**
   * 发送消息（REST）：私聊传 { receiverId }，群聊传 { roomId }。
   * @returns {Promise<object>} 后端保存的消息对象
   */
  async sendMessage({ receiverId, roomId, content, type = 'text' }) {
    const res = await messageApi.sendMessage({
      receiver_id: receiverId,
      room_id: roomId,
      content,
      type,
    });
    // 发送者自己的消息即时上屏（服务端也会向其他客户端推送）
    this.appendLocalMessage(res.data);
    return res.data;
  },

  /**
   * 加载私聊消息：拉取与目标用户的往来历史，替换当前消息列表并标记已读。
   */
  async loadPrivateMessages(userId) {
    const res = await messageApi.getMessages({ target_id: userId, page: 1, page_size: 100 });
    setState({ messages: res.data.items || [] });
    await this.markAllRead(res.data.items || []);
    return res.data.items || [];
  },

  /**
   * 加载群聊消息：拉取房间历史，替换当前消息列表并标记已读。
   */
  async loadRoomMessages(roomId) {
    const res = await messageApi.getMessages({ room_id: roomId, page: 1, page_size: 100 });
    setState({ messages: res.data.items || [] });
    await this.markAllRead(res.data.items || []);
    return res.data.items || [];
  },

  /** 标记单条消息已读（仅接收者本人） */
  async markAsRead(messageId) {
    return messageApi.markAsRead(messageId);
  },

  /** 拉取未读消息数并写入 store（顶栏角标） */
  async loadUnreadCount() {
    try {
      const res = await messageApi.getUnreadCount();
      setState({ unreadMessages: res.data.unread_count || 0 });
    } catch {
      /* 未读数为非关键数据，失败静默 */
    }
  },

  /** 批量标记已读（切换到某会话后调用） */
  async markAllRead(messages) {
    const { user } = getState();
    const unread = (messages || []).filter((m) => m.status !== 'read' && m.receiver_id === user?.id);
    await Promise.all(unread.map((m) => this.markAsRead(m.id).catch(() => {})));
    await this.loadUnreadCount();
  },

  /**
   * 处理 WebSocket 实时消息：
   *   - 属于当前会话 → 追加到列表
   *   - 其他会话 → 未读数+1，并转一条站内通知
   */
  handleRealtimeMessage(msg) {
    const { user, activeRoom, activeUser, messages } = getState();
    if (!user) return;

    const isGroup = !!msg.room_id;
    const isCurrentGroup = isGroup && activeRoom && activeRoom.id === msg.room_id;
    const isCurrentPrivate = !isGroup && activeUser && (msg.sender_id === activeUser.id || msg.sender_id === user.id);
    const isSelf = msg.sender_id === user.id;

    if (isCurrentGroup || isCurrentPrivate || isSelf) {
      // 追加并去重（同一消息可能因 REST 回显与 WS 推送各到一次）
      if (!messages.some((m) => m.id === msg.id)) {
        setState({ messages: [...messages, msg] });
      }
      return;
    }

    // 非当前会话：累计未读，并产生通知
    if (!isSelf) {
      setState({ unreadMessages: getState().unreadMessages + 1 });
      notificationService.emitNotification({
        title: isGroup ? '群聊新消息' : '新私聊消息',
        content: msg.content,
      });
    }
  },

  /** 本地追加消息（发送成功后回显，与实时消息去重逻辑共用） */
  appendLocalMessage(msg) {
    const { messages } = getState();
    if (!messages.some((m) => m.id === msg.id)) {
      setState({ messages: [...messages, msg] });
    }
  },

  /** 错误提示统一入口（供页面 catch 后调用） */
  showError(err, fallback = '操作失败') {
    toast.show(err.message || fallback, 'error');
  },
};
