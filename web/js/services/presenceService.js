/**
 * 在线状态业务服务：在线用户列表与单用户状态查询，维护 store.onlineUsers。
 */
import { presenceApi } from '../api/presence.js';
import { setState } from '../store/index.js';
import { getState } from '../store/index.js';

export const presenceService = {
  /** 拉取在线用户 ID 列表（配合用户资料可渲染在线用户侧栏） */
  async getOnlineUsers() {
    const res = await presenceApi.getOnlineUsers();
    setState({ onlineUsers: res.data || [] });
    return res.data || [];
  },

  /** 查询单个用户在线状态 */
  async getPresence(userId) {
    const res = await presenceApi.getPresence(userId);
    return res.data;
  },

  /** 处理 WS presence 广播：更新在线用户集合 */
  handlePresenceBroadcast(data) {
    const { onlineUsers } = getState();
    const set = new Set(onlineUsers);
    if (data.status === 'offline') set.delete(data.user_id);
    else set.add(data.user_id);
    setState({ onlineUsers: Array.from(set) });
  },
};
