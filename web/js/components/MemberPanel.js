/**
 * MemberPanel 侧栏面板组件：三态渲染——
 *   1. 群聊会话：房间成员列表（含在线状态、群主标识），点击成员可发起私聊
 *   2. 私聊会话：对方资料卡（头像/昵称/在线状态）
 *   3. 无会话：全部在线用户列表（点击发起私聊）
 * 订阅 store.roomMembers / onlineUsers / activeUser / activeRoom。
 */
import { createEl, escapeHtml } from '../utils/dom.js';
import { getState, subscribe } from '../store/index.js';
import { createAvatar } from './Avatar.js';

/**
 * @param {object} options
 * @param {Function} [options.onInvite] () => void 群聊邀请按钮回调
 * @param {Function} [options.onSelectMember] (user) => void 点击群成员（发起私聊）
 * @param {Function} [options.onSelectUser] (user) => void 点击在线用户（发起私聊）
 */
export function createMemberPanel({ onInvite, onSelectMember, onSelectUser } = {}) {
  const el = createEl('div', 'member-panel');

  /** 群成员表 → userId → 昵称映射（昵称在成员关联的 user 对象上） */
  function nicknameOf(userId) {
    const { roomMembers } = getState();
    const member = roomMembers.find((m) => m.user_id === userId);
    return member ? (member.user?.nickname || `用户${userId}`) : `用户${userId}`;
  }

  function render() {
    const { roomMembers, activeUser, activeRoom, user, onlineUsers } = getState();
    el.innerHTML = '';

    // 私聊场景：对方资料卡
    if (activeUser && !activeRoom) {
      const card = createEl('div', 'member-card', el);
      const avatarWrap = createEl('div', 'member-card-avatar', card);
      avatarWrap.appendChild(createAvatar({ nickname: activeUser.nickname, avatar: activeUser.avatar }, 64));
      card.innerHTML += `
        <div class="member-card-name">${escapeHtml(activeUser.nickname || `用户${activeUser.id}`)}</div>
        <div class="member-card-status">${onlineUsers.includes(activeUser.id) ? '在线' : '离线'}</div>
      `;
      return;
    }

    // 群聊场景：成员列表（点击成员 → 私聊）
    if (activeRoom) {
      const header = createEl('div', 'member-panel-header');
      header.innerHTML = `<span>成员（${roomMembers.length}）</span>`;
      if (onInvite) {
        const inviteBtn = createEl('button', 'btn btn-sm btn-ghost', header);
        inviteBtn.textContent = '邀请';
        inviteBtn.addEventListener('click', onInvite);
      }
      el.appendChild(header);

      if (!roomMembers.length) {
        const tip = createEl('div', 'empty-tip', el);
        tip.textContent = '暂无成员';
        return;
      }
      roomMembers.forEach((member) => {
        const li = createEl('div', 'list-item member-item', el);
        li.appendChild(createAvatar({ nickname: member.user?.nickname, avatar: member.user?.avatar }, 32));
        const info = createEl('div', 'member-item-info', li);
        const isOwner = member.role === 'owner';
        const isMe = user && member.user_id === user.id;
        info.innerHTML = `
          <div class="text-ellipsis">${escapeHtml(member.user?.nickname || `用户${member.user_id}`)}${isMe ? '（我）' : ''}</div>
          <div class="member-item-meta text-ellipsis">${onlineUsers.includes(member.user_id) ? '在线' : '离线'}${isOwner ? ' · 群主' : ''}</div>
        `;
        if (!isMe) {
          li.addEventListener('click', () =>
            onSelectMember &&
            onSelectMember({
              id: member.user_id,
              nickname: member.user?.nickname,
              avatar: member.user?.avatar,
            })
          );
        }
      });
      return;
    }

    // 无会话：在线用户列表（供发起私聊）
    const header = createEl('div', 'member-panel-header');
    header.innerHTML = `<span>在线用户（${onlineUsers.filter((id) => id !== (user && user.id)).length}）</span>`;
    el.appendChild(header);

    const others = onlineUsers.filter((id) => id !== (user && user.id));
    if (!others.length) {
      const tip = createEl('div', 'empty-tip', el);
      tip.textContent = '暂无其他在线用户';
      return;
    }
    others.forEach((userId) => {
      const li = createEl('div', 'list-item member-item', el);
      li.appendChild(createAvatar({ id: userId, nickname: nicknameOf(userId) }, 32));
      const info = createEl('div', 'member-item-info', li);
      info.innerHTML = `
        <div class="text-ellipsis">${escapeHtml(nicknameOf(userId))}</div>
        <div class="member-item-meta">在线 · 点击私聊</div>
      `;
      li.addEventListener('click', () =>
        onSelectUser && onSelectUser({ id: userId, nickname: nicknameOf(userId) })
      );
    });
  }

  // 订阅多类状态变化：成员、在线列表、会话切换
  const unsubs = [
    subscribe('roomMembers', render),
    subscribe('onlineUsers', render),
    subscribe('activeUser', render),
    subscribe('activeRoom', render),
  ];
  render();

  return { el, destroy: () => unsubs.forEach((u) => u()) };
}
