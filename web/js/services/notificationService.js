/**
 * 通知业务服务：列表/已读/未读计数，以及 WebSocket 实时通知的归置。
 */
import { notificationApi } from '../api/notification.js';
import { setState } from '../store/index.js';
import { getState } from '../store/index.js';
import { toast } from '../components/Toast.js';

export const notificationService = {
  /** 拉取通知列表并同步未读计数 */
  async getNotifications() {
    const res = await notificationApi.getNotifications({ page: 1, page_size: 50 });
    const items = res.data.items || [];
    setState({
      notifications: items,
      unreadNotifications: items.filter((n) => !n.is_read).length,
    });
    return items;
  },

  /** 标记单条通知已读并刷新列表 */
  async markAsRead(id) {
    await notificationApi.markAsRead(id);
    await this.getNotifications();
  },

  /** 标记全部通知已读（遍历未读项） */
  async markAllRead() {
    const { notifications } = getState();
    const unread = notifications.filter((n) => !n.is_read);
    await Promise.all(unread.map((n) => notificationApi.markAsRead(n.id).catch(() => {})));
    await this.getNotifications();
  },

  /**
   * 处理 WebSocket 实时通知：追加到列表并弹出轻提示。
   * 同时被 messageService 复用（离线消息转通知场景）。
   */
  handleRealtimeNotification(notification) {
    const { notifications } = getState();
    setState({
      notifications: [notification, ...notifications],
      unreadNotifications: getState().unreadNotifications + 1,
    });
    toast.show(`${notification.title}：${notification.content || ''}`, 'info');
  },

  /** 生成一条本地通知（不落库，用于非当前会话消息的即时提示） */
  emitNotification({ title, content }) {
    this.handleRealtimeNotification({
      id: `local_${Date.now()}`,
      title,
      content,
      is_read: false,
      created_at: new Date().toISOString(),
    });
  },
};
