/**
 * 全局状态中心：极简的订阅-发布实现，替代框架状态库。
 * 约定：
 *   - 所有跨组件共享的数据统一放 state，禁止在组件间直接传递私有数据
 *   - 修改状态只能通过 setState 打补丁（浅合并），修改后自动通知订阅者
 *   - 组件按需订阅字段，key 为 '*' 时订阅全部变更
 * 分层：views / components 只读 state 并订阅变化，写操作经由 services 层完成。
 */

const state = {
  /** 登录态 */
  token: '',
  user: null,

  /** WebSocket 连接状态 */
  wsConnected: false,

  /** 当前会话：群聊 activeRoom={id,name,type} / 私聊 activeUser={id,nickname,avatar,status} */
  activeRoom: null,
  activeUser: null,

  /** 房间列表与当前房间成员 */
  rooms: [],
  roomMembers: [],

  /** 当前会话消息列表（新消息追加、切换会话时整体替换） */
  messages: [],

  /** 通知列表与未读计数 */
  notifications: [],
  unreadNotifications: 0,

  /** 在线用户 ID 集合（来自在线列表接口 + presence 广播） */
  onlineUsers: [],

  /** 当前用户未读私聊消息数 */
  unreadMessages: 0,
};

const listeners = new Map(); // key -> Set<fn>

/** 获取状态（只读约定：调用方不得直接修改返回对象） */
export function getState() {
  return state;
}

/** 合并补丁更新状态并通知订阅者（先通知对应字段，再通知全量） */
export function setState(patch) {
  Object.assign(state, patch);
  Object.keys(patch).forEach((key) => notify(key));
  notify('*');
}

/** 订阅状态变更；返回取消订阅函数 */
export function subscribe(key, fn) {
  if (!listeners.has(key)) listeners.set(key, new Set());
  const set = listeners.get(key);
  set.add(fn);
  return () => set.delete(fn);
}

/** 通知指定 key 的订阅者（同时通知全量订阅者） */
export function notify(key) {
  (listeners.get(key) || new Set()).forEach((fn) => fn());
  (listeners.get('*') || new Set()).forEach((fn) => fn());
}

/** 重置登录相关状态（登出/401 时调用） */
export function resetAuth() {
  Object.assign(state, {
    token: '',
    user: null,
    wsConnected: false,
    activeRoom: null,
    activeUser: null,
    rooms: [],
    roomMembers: [],
    messages: [],
    notifications: [],
    unreadNotifications: 0,
    onlineUsers: [],
    unreadMessages: 0,
  });
  notify('*');
}
