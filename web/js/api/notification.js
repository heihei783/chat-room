/**
 * 通知模块接口（均需 JWT）。
 */
import { http } from './http.js';

export const notificationApi = {
  /** 查询当前用户通知列表，支持分页 */
  getNotifications: (params) => http.get('/api/notifications', params),

  /** 标记通知已读（仅接收者本人） */
  markAsRead: (id) => http.post(`/api/notifications/${id}/read`),
};
