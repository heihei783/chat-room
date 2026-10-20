/**
 * MessageList 消息列表组件：渲染当前会话消息（气泡式），自动滚动到底部。
 * 订阅 store.messages / user / roomMembers：新消息、自身资料变化、成员头像变化时自动重渲染。
 */
import { createEl, escapeHtml } from '../utils/dom.js';
import { getState, subscribe } from '../store/index.js';
import { formatTime } from '../utils/format.js';
import { createAvatar } from './Avatar.js';
import { resolveAssetUrl } from '../config/env.js';

export function createMessageList() {
  const el = createEl('div', 'message-list');

  /**
   * 由 sender_id 解析发送者展示信息（昵称 + 头像）。
   * 头像优先级：自己的资料 user.avatar → 私聊对象 activeUser.avatar → 群成员 member.user.avatar。
   */
  function senderInfo(senderId) {
    const { user, activeUser, roomMembers } = getState();
    if (senderId === user?.id) return { name: '', avatar: user?.avatar || '' };
    if (activeUser && senderId === activeUser.id) {
      return { name: activeUser.nickname || '对方', avatar: activeUser.avatar || '' };
    }
    // 成员接口返回 { user_id, role, user: { nickname, avatar } }，昵称与头像在 user 关联对象上
    const member = roomMembers.find((m) => m.user_id === senderId);
    if (member) return { name: member.user?.nickname || `用户${senderId}`, avatar: member.user?.avatar || '' };
    return { name: `用户${senderId}`, avatar: '' };
  }

  /** 渲染单条消息内容（按 type 区分展示） */
  function renderContent(msg) {
    if (msg.type === 'image') {
      return `<img class="msg-image" src="${escapeHtml(resolveAssetUrl(msg.content))}" alt="图片消息" loading="lazy">`;
    }
    if (msg.type === 'file') {
      return `<a class="msg-file" href="${escapeHtml(resolveAssetUrl(msg.content))}" target="_blank" rel="noopener">📎 文件下载</a>`;
    }
    return `<span class="msg-text">${escapeHtml(msg.content)}</span>`;
  }

  function render() {
    const { messages, user } = getState();
    el.innerHTML = '';
    if (!messages.length) {
      const tip = createEl('div', 'empty-tip', el);
      tip.textContent = '还没有消息，说点什么吧';
      return;
    }
    messages.forEach((msg) => {
      const isSelf = user && msg.sender_id === user.id;
      const row = createEl('div', `msg-row ${isSelf ? 'msg-self' : 'msg-other'}`, el);
      const { name, avatar } = senderInfo(msg.sender_id);
      row.appendChild(createAvatar({ id: msg.sender_id, nickname: name || '我', avatar }, 36));
      const body = createEl('div', 'msg-body', row);
      if (name) body.innerHTML += `<div class="msg-sender">${escapeHtml(name)}</div>`;
      const bubble = createEl('div', 'msg-bubble', body);
      bubble.innerHTML = renderContent(msg);
      const meta = createEl('div', 'msg-meta', body);
      meta.textContent = formatTime(msg.created_at);
    });
    scrollToBottom();
  }

  function scrollToBottom() {
    el.scrollTop = el.scrollHeight;
  }

  // 消息、自身资料、成员头像任一变化都重渲染
  const unsubs = [
    subscribe('messages', render),
    subscribe('user', render),
    subscribe('roomMembers', render),
  ];
  render();

  return { el, scrollToBottom, destroy: () => unsubs.forEach((u) => u()) };
}
