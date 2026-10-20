/**
 * RoomList 房间列表组件：渲染当前用户加入的房间，支持选择会话、创建、加入。
 * 订阅 store.rooms 与 activeRoom，点击房间回调 onSelect。
 */
import { createEl, escapeHtml } from '../utils/dom.js';
import { getState, subscribe } from '../store/index.js';
import { createAvatar } from './Avatar.js';

/**
 * @param {object} options
 * @param {Function} options.onSelect (room) => void 点击房间
 * @param {Function} options.onCreate () => void 点击创建房间
 * @param {Function} options.onJoin () => void 点击加入房间（按 ID 加入）
 */
export function createRoomList({ onSelect, onCreate, onJoin } = {}) {
  const el = createEl('div', 'room-list');

  /** 头部：标题 + 加入/创建按钮 */
  const header = createEl('div', 'room-list-header');
  header.innerHTML = `<span class="room-list-title">房间列表</span>`;
  const actions = createEl('div', 'room-list-actions flex gap-8', header);
  if (onJoin) {
    const joinBtn = createEl('button', 'btn btn-sm btn-ghost', actions);
    joinBtn.textContent = '加入';
    joinBtn.addEventListener('click', onJoin);
  }
  if (onCreate) {
    const createBtn = createEl('button', 'btn btn-sm btn-ghost', actions);
    createBtn.textContent = '+ 创建';
    createBtn.addEventListener('click', onCreate);
  }
  el.appendChild(header);

  const list = createEl('ul', 'room-list-items', el);

  /** 渲染列表（每次全量重建，数据量小、保证一致性） */
  function render() {
    const { rooms, activeRoom } = getState();
    list.innerHTML = '';
    if (!rooms.length) {
      const tip = createEl('li', 'empty-tip', list);
      tip.textContent = '暂无房间，点击右上角创建或加入';
      return;
    }
    rooms.forEach((room) => {
      const li = createEl('li', 'list-item room-item', list);
      if (activeRoom && activeRoom.id === room.id) li.classList.add('active');
      li.appendChild(createAvatar({ nickname: room.name, avatar: room.avatar }, 36));
      const info = createEl('div', 'room-item-info', li);
      info.innerHTML = `
        <div class="room-item-name text-ellipsis">${escapeHtml(room.name)}</div>
        <div class="room-item-meta text-ellipsis">${room.member_count || 0} 位成员</div>
      `;
      li.addEventListener('click', () => onSelect && onSelect(room));
    });
  }

  // 订阅变化自动刷新
  const unsubscribe = subscribe('rooms', render);
  render();

  return { el, destroy: unsubscribe };
}
