/**
 * NotificationCenter 通知中心组件：顶部铃铛（未读角标）+ 下拉通知列表。
 * 订阅 store.notifications 与 unreadNotifications；支持标记全部已读。
 */
import { createEl, escapeHtml } from '../utils/dom.js';
import { getState, subscribe } from '../store/index.js';
import { formatTime } from '../utils/format.js';
import { notificationService } from '../services/notificationService.js';

export function createNotificationCenter() {
  const el = createEl('div', 'notification-center');

  /** 铃铛按钮 + 未读角标 */
  const bell = createEl('button', 'btn btn-ghost btn-sm', el);
  bell.innerHTML = '🔔';
  const badge = createEl('span', 'notification-badge', bell);

  /** 下拉面板（点击铃铛切换显示） */
  const panel = createEl('div', 'notification-panel', el);
  panel.style.display = 'none';

  const panelHeader = createEl('div', 'notification-panel-header', panel);
  panelHeader.innerHTML = '<span>通知</span>';
  const readAllBtn = createEl('button', 'btn btn-sm btn-ghost', panelHeader);
  readAllBtn.textContent = '全部已读';
  readAllBtn.addEventListener('click', async () => {
    await notificationService.markAllRead();
  });

  const list = createEl('div', 'notification-list', panel);

  bell.addEventListener('click', (e) => {
    e.stopPropagation();
    panel.style.display = panel.style.display === 'none' ? 'block' : 'none';
  });
  document.addEventListener('click', (e) => {
    if (!el.contains(e.target)) panel.style.display = 'none';
  });

  /** 渲染角标与列表 */
  function render() {
    const { notifications, unreadNotifications } = getState();
    badge.textContent = unreadNotifications || '';
    badge.style.display = unreadNotifications ? 'inline-flex' : 'none';

    list.innerHTML = '';
    if (!notifications.length) {
      const tip = createEl('div', 'empty-tip', list);
      tip.textContent = '暂无通知';
      return;
    }
    notifications.forEach((n) => {
      const item = createEl('div', `notification-item ${n.is_read ? '' : 'notification-unread'}`, list);
      item.innerHTML = `
        <div class="notification-title">${escapeHtml(n.title || '通知')}</div>
        <div class="notification-content text-ellipsis">${escapeHtml(n.content || '')}</div>
        <div class="notification-time">${formatTime(n.created_at)}</div>
      `;
      item.addEventListener('click', async () => {
        if (!n.is_read) {
          await notificationService.markAsRead(n.id);
        }
      });
    });
  }

  const unsubscribe = subscribe('notifications', render);
  render();

  return { el, refresh: () => notificationService.getNotifications(), destroy: unsubscribe };
}
