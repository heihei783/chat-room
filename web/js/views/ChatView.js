/**
 * ChatView 聊天主页面：三栏布局（房间列表 / 会话区 / 成员在线面板）。
 * 职责：
 *   - 初始化：拉取用户资料、房间、在线列表、通知、未读数，建立 WebSocket
 *   - 会话管理：群聊/私聊切换、加入/创建/退出房间、发起私聊
 *   - 消息收发：文本与图片消息，实时消息由 messageService 归置后自动渲染
 * 所有页面数据均经 services 层读写 store，视图只做组装与事件绑定。
 */
import { $el, escapeHtml } from '../utils/dom.js';
import * as storage from '../utils/storage.js';
import { getState, setState, subscribe } from '../store/index.js';
import { openModal, confirmDialog } from '../components/Modal.js';
import { toast } from '../components/Toast.js';
import { createAvatar } from '../components/Avatar.js';
import { createRoomList } from '../components/RoomList.js';
import { createMessageList } from '../components/MessageList.js';
import { createMessageInput } from '../components/MessageInput.js';
import { createMemberPanel } from '../components/MemberPanel.js';
import { createNotificationCenter } from '../components/NotificationCenter.js';
import { authService } from '../services/authService.js';
import { userService } from '../services/userService.js';
import { roomService } from '../services/roomService.js';
import { messageService } from '../services/messageService.js';
import { presenceService } from '../services/presenceService.js';
import { notificationService } from '../services/notificationService.js';
import { uploadService } from '../services/uploadService.js';
import { wsClient } from '../websocket/index.js';

export const ChatView = {
  title: '聊天',

  render(container) {
    container.innerHTML = `
      <div class="chat-layout">
        <!-- 左侧栏：当前用户 + 房间列表 -->
        <aside class="chat-sidebar">
          <div class="sidebar-user">
            <div class="flex items-center gap-8 flex-1" style="min-width:0">
              <span id="sidebar-avatar"></span>
              <div class="flex-1" style="min-width:0">
                <div class="sidebar-nickname text-ellipsis" id="sidebar-nickname">--</div>
                <div class="sidebar-status" id="sidebar-status">连接中…</div>
              </div>
            </div>
            <div class="flex items-center gap-8">
              <button class="btn btn-sm btn-ghost" id="edit-profile-btn">编辑</button>
              <button class="btn btn-sm btn-ghost" id="logout-btn">退出</button>
            </div>
          </div>
          <div class="sidebar-rooms" id="sidebar-list"></div>
        </aside>

        <!-- 中间：会话区 -->
        <main class="chat-main">
          <header class="chat-header">
            <div class="flex-1" style="min-width:0">
              <div class="chat-title text-ellipsis" id="chat-title">选择一个会话</div>
              <div class="chat-meta" id="chat-meta">--</div>
            </div>
            <div class="flex items-center gap-8">
              <button class="btn btn-sm btn-ghost" id="leave-btn" style="display:none">退出群聊</button>
              <div id="header-notify"></div>
            </div>
          </header>
          <div class="chat-messages" id="message-area"></div>
          <div class="chat-input" id="input-area"></div>
        </main>

        <!-- 右侧：成员 / 在线面板 -->
        <aside class="chat-members" id="member-area"></aside>
      </div>
    `;

    /** ---------- 组件挂载 ---------- */
    const roomList = createRoomList({
      onSelect: selectRoom,
      onCreate: openCreateRoom,
      onJoin: openJoinRoom,
    });
    $el('#sidebar-list', container).appendChild(roomList.el);

    const messageList = createMessageList();
    $el('#message-area', container).appendChild(messageList.el);

    const messageInput = createMessageInput({ onSendText: sendText, onSendImage: sendImage });
    $el('#input-area', container).appendChild(messageInput.el);

    const memberPanel = createMemberPanel({
      onInvite: openInvite,
      onSelectMember: selectUser,
      onSelectUser: selectUser,
    });
    $el('#member-area', container).appendChild(memberPanel.el);

    const notificationCenter = createNotificationCenter();
    $el('#header-notify', container).appendChild(notificationCenter.el);

    $el('#logout-btn', container).addEventListener('click', logout);
    $el('#edit-profile-btn', container).addEventListener('click', openEditProfile);
    $el('#leave-btn', container).addEventListener('click', leaveRoom);

    /** ---------- 初始化数据 ---------- */
    if (!getState().user) userService.getProfile().catch(() => {});
    roomService.getRooms().catch((err) => toast.show(err.message, 'error'));
    presenceService.getOnlineUsers().catch(() => {});
    notificationService.getNotifications().catch(() => {});
    messageService.loadUnreadCount();
    wsClient.connect();

    /** ---------- 会话切换 ---------- */
    async function selectRoom(room) {
      if (getState().activeRoom && getState().activeRoom.id === room.id) return;
      // 订阅群聊推送（join 幂等，重复发送无害）
      wsClient.joinRoom(room.id);
      setState({ activeRoom: room, activeUser: null, messages: [] });
      // 持久化当前会话：刷新页面后自动恢复
      storage.set('last_session', {
        type: 'room',
        id: room.id,
        name: room.name,
        avatar: room.avatar,
        member_count: room.member_count,
      });
      roomService.getMembers(room.id).catch(() => {});
      messageService.loadRoomMessages(room.id).catch((err) => toast.show(err.message, 'error'));
    }

    async function selectUser(user) {
      if (getState().activeUser && getState().activeUser.id === user.id) return;
      setState({ activeUser: user, activeRoom: null, messages: [] });
      // 持久化当前会话：刷新页面后自动恢复
      storage.set('last_session', {
        type: 'user',
        id: user.id,
        nickname: user.nickname,
        avatar: user.avatar,
      });
      messageService.loadPrivateMessages(user.id).catch((err) => toast.show(err.message, 'error'));
    }

    /** 恢复上次会话：刷新页面后自动回到原会话并加载历史消息 */
    async function restoreSession() {
      const saved = storage.get('last_session');
      if (!saved) return;
      if (saved.type === 'room') {
        try {
          // 以最新房间数据恢复（房间可能已被删除/移出成员，失败则清除记录）
          const room = await roomService.getRoom(saved.id);
          selectRoom(room);
        } catch {
          storage.remove('last_session');
        }
      } else if (saved.type === 'user') {
        selectUser({ id: saved.id, nickname: saved.nickname, avatar: saved.avatar });
      }
    }

    /** ---------- 消息发送 ---------- */
    async function sendText(text) {
      const { activeRoom, activeUser } = getState();
      if (!activeRoom && !activeUser) return toast.show('请先选择会话', 'warning');
      messageInput.setSending(true);
      try {
        if (activeRoom) {
          await messageService.sendMessage({ roomId: activeRoom.id, content: text });
        } else {
          await messageService.sendMessage({ receiverId: activeUser.id, content: text });
        }
      } catch (err) {
        toast.show(err.message || '发送失败', 'error');
      } finally {
        messageInput.setSending(false);
      }
    }

    async function sendImage(url) {
      const { activeRoom, activeUser } = getState();
      if (!activeRoom && !activeUser) return toast.show('请先选择会话', 'warning');
      try {
        if (activeRoom) {
          await messageService.sendMessage({ roomId: activeRoom.id, content: url, type: 'image' });
        } else {
          await messageService.sendMessage({ receiverId: activeUser.id, content: url, type: 'image' });
        }
      } catch (err) {
        toast.show(err.message || '图片发送失败', 'error');
      }
    }

    /** ---------- 房间操作 ---------- */
    /** 创建房间弹窗 */
    function openCreateRoom() {
      const body = document.createElement('div');
      body.innerHTML = `
        <div class="form-group">
          <label>房间名称</label>
          <input class="form-input" id="modal-room-name" placeholder="必填，最多 30 字">
        </div>
        <div class="form-group">
          <label>房间描述（选填）</label>
          <input class="form-input" id="modal-room-desc" placeholder="一句话介绍房间">
        </div>
        <div class="form-group">
          <label>类型</label>
          <select class="form-input" id="modal-room-type">
            <option value="group">群聊</option>
            <option value="channel">频道</option>
          </select>
        </div>
      `;
      openModal({
        title: '创建房间',
        body,
        okText: '创建',
        onOk: async () => {
          const name = $el('#modal-room-name', body).value.trim();
          if (!name) {
            toast.show('请输入房间名称', 'warning');
            return false;
          }
          const description = $el('#modal-room-desc', body).value.trim();
          const type = $el('#modal-room-type', body).value;
          try {
            const room = await roomService.createRoom({ name, description, type });
            toast.show('房间创建成功', 'success');
            selectRoom(room);
          } catch (err) {
            toast.show(err.message || '创建失败', 'error');
            return false;
          }
        },
      });
    }

    /** 加入房间弹窗（按房间 ID） */
    function openJoinRoom() {
      const body = document.createElement('div');
      body.innerHTML = `
        <div class="form-group">
          <label>房间 ID</label>
          <input class="form-input" id="modal-room-id" placeholder="输入房间 ID">
        </div>
      `;
      openModal({
        title: '加入房间',
        body,
        okText: '加入',
        onOk: async () => {
          const id = Number($el('#modal-room-id', body).value.trim());
          if (!id) {
            toast.show('请输入房间 ID', 'warning');
            return false;
          }
          try {
            await roomService.joinRoom(id);
            const room = await roomService.getRoom(id);
            toast.show('已加入房间', 'success');
            selectRoom(room);
          } catch (err) {
            toast.show(err.message || '加入失败', 'error');
            return false;
          }
        },
      });
    }

    /** 退出当前群聊 */
    function leaveRoom() {
      const { activeRoom } = getState();
      if (!activeRoom) return;
      confirmDialog('退出群聊', `确定退出「${escapeHtml(activeRoom.name)}」吗？退出后无法收到该房间消息。`, async () => {
        try {
          await roomService.leaveRoom(activeRoom.id);
          toast.show('已退出群聊', 'success');
        } catch (err) {
          toast.show(err.message || '退出失败', 'error');
        }
      });
    }

    /** 邀请加入：分享房间 ID */
    function openInvite() {
      const { activeRoom } = getState();
      if (!activeRoom) return;
      const body = document.createElement('div');
      body.innerHTML = `
        <p class="text-secondary">将房间 ID 分享给好友，对方登录后在房间列表点击「加入」输入即可。</p>
        <p style="margin-top:12px">房间 ID：<b id="invite-room-id">${activeRoom.id}</b></p>
      `;
      openModal({
        title: '邀请加入',
        body,
        showOk: false,
      });
      // 点击 ID 复制
      const idEl = $el('#invite-room-id', body);
      idEl.style.cursor = 'pointer';
      idEl.addEventListener('click', async () => {
        try {
          await navigator.clipboard.writeText(String(activeRoom.id));
          toast.show('房间 ID 已复制', 'success');
        } catch {
          toast.show('复制失败，请手动复制', 'error');
        }
      });
    }

    /** ---------- 用户操作 ---------- */
    function logout() {
      storage.remove('last_session');
      authService.logout();
      location.hash = '#/login';
    }

    /**
     * 编辑资料弹窗：修改昵称与头像。
     * 头像先经 uploadService 上传获取 URL，保存时随昵称一并提交 PUT /api/user/profile。
     */
    function openEditProfile() {
      const { user } = getState();
      if (!user) return;

      const body = document.createElement('div');
      body.innerHTML = `
        <div class="form-group">
          <label>昵称</label>
          <input class="form-input" id="edit-nickname" maxlength="50" placeholder="请输入昵称">
        </div>
        <div class="form-group">
          <label>头像</label>
          <div class="flex items-center gap-8">
            <span id="edit-avatar-preview"></span>
            <button class="btn btn-sm btn-ghost" id="edit-avatar-btn" type="button">选择图片</button>
          </div>
          <input type="file" id="edit-avatar-file" accept="image/*" style="display:none">
          <div class="text-secondary" style="font-size:12px;margin-top:6px">支持 jpg/png/gif，最大 10MB</div>
        </div>
      `;

      $el('#edit-nickname', body).value = user.nickname || user.username || '';
      const previewWrap = $el('#edit-avatar-preview', body);
      previewWrap.appendChild(createAvatar(user, 48));

      // 选择图片 → 上传 → 预览新头像（保存时生效）
      const fileInput = $el('#edit-avatar-file', body);
      let avatarUrl = '';
      $el('#edit-avatar-btn', body).addEventListener('click', () => fileInput.click());
      fileInput.addEventListener('change', async () => {
        const file = fileInput.files && fileInput.files[0];
        fileInput.value = '';
        if (!file) return;
        try {
          avatarUrl = await uploadService.upload(file);
          previewWrap.innerHTML = '';
          previewWrap.appendChild(createAvatar({ nickname: 'new', avatar: avatarUrl }, 48));
          toast.show('头像已上传，点击「保存」生效', 'success');
        } catch (err) {
          toast.show(err.message || '头像上传失败', 'error');
        }
      });

      openModal({
        title: '编辑资料',
        body,
        okText: '保存',
        onOk: async () => {
          const nickname = $el('#edit-nickname', body).value.trim();
          const data = {};
          if (nickname && nickname !== (user.nickname || '')) data.nickname = nickname;
          if (avatarUrl && avatarUrl !== user.avatar) data.avatar = avatarUrl;
          if (!data.nickname && !data.avatar) {
            toast.show('请修改昵称或头像后再保存', 'warning');
            return false;
          }
          try {
            await userService.updateProfile(data);
            // 资料变更后刷新当前房间成员列表，同步成员面板与消息列表中的头像/昵称
            const { activeRoom } = getState();
            if (activeRoom) roomService.getMembers(activeRoom.id).catch(() => {});
            toast.show('资料已更新', 'success');
          } catch (err) {
            toast.show(err.message || '保存失败', 'error');
            return false;
          }
        },
      });
    }

    /** ---------- 顶栏与用户信息渲染 ---------- */
    function renderHeader() {
      const { activeRoom, activeUser, onlineUsers, wsConnected } = getState();
      const title = $el('#chat-title', container);
      const meta = $el('#chat-meta', container);
      const leaveBtn = $el('#leave-btn', container);
      if (activeRoom) {
        title.textContent = activeRoom.name;
        meta.textContent = `${activeRoom.member_count || 0} 位成员`;
        leaveBtn.style.display = '';
      } else if (activeUser) {
        title.textContent = activeUser.nickname || `用户${activeUser.id}`;
        meta.textContent = onlineUsers.includes(activeUser.id) ? '在线' : '离线';
        leaveBtn.style.display = 'none';
      } else {
        title.textContent = '选择一个会话';
        meta.textContent = wsConnected ? '连接正常' : '连接中…';
        leaveBtn.style.display = 'none';
      }
    }

    function renderUser() {
      const { user, wsConnected } = getState();
      const avatarWrap = $el('#sidebar-avatar', container);
      const nickname = $el('#sidebar-nickname', container);
      const status = $el('#sidebar-status', container);
      avatarWrap.innerHTML = '';
      if (user) {
        avatarWrap.appendChild(createAvatar(user, 36));
        nickname.textContent = user.nickname || user.username;
        status.textContent = wsConnected ? '在线' : '连接中…';
      }
    }

    renderHeader();
    renderUser();

    // 恢复上次会话（刷新后自动回到原会话并加载历史消息）
    restoreSession();

    // 会话/连接状态变化时刷新顶栏与用户信息
    const unsubs = [
      subscribe('activeRoom', renderHeader),
      subscribe('activeUser', renderHeader),
      subscribe('onlineUsers', renderHeader),
      subscribe('wsConnected', () => {
        renderHeader();
        renderUser();
      }),
      subscribe('user', renderUser),
    ];

    // 视图销毁时取消订阅（路由切换时由 router 调用）
    container.__viewCleanup = () => {
      unsubs.forEach((u) => u());
      roomList.destroy();
      messageList.destroy();
      memberPanel.destroy();
      notificationCenter.destroy();
    };
  },
};
