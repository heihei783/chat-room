/**
 * 消息模块接口（均需 JWT）。
 */
import { http } from './http.js';

export const messageApi = {
  /** 发送消息：私聊传 receiver_id，群聊传 room_id（二选一） */
  sendMessage: (data) => http.post('/api/messages', data),

  /** 查询消息列表：target_id=私聊对象 / room_id=群聊房间，支持分页 */
  getMessages: (params) => http.get('/api/messages', params),

  /** 查询未读消息数 */
  getUnreadCount: () => http.get('/api/messages/unread-count'),

  /** 标记消息已读（仅接收者本人） */
  markAsRead: (id) => http.post(`/api/messages/${id}/read`),
};
