/**
 * 在线状态模块接口（均需 JWT）。
 */
import { http } from './http.js';

export const presenceApi = {
  /** 获取在线用户 ID 列表（Redis 实时状态） */
  getOnlineUsers: () => http.get('/api/users/online'),

  /** 查询指定用户在线状态 */
  getPresence: (userId) => http.get(`/api/users/${userId}/presence`),
};
