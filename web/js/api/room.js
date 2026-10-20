/**
 * 房间模块接口（均需 JWT）。
 */
import { http } from './http.js';

export const roomApi = {
  /** 创建房间：name 必填，description/type/avatar 可选 */
  createRoom: (data) => http.post('/api/rooms', data),

  /** 查询当前用户加入的房间列表 */
  getRooms: () => http.get('/api/rooms'),

  /** 查询房间详情 */
  getRoom: (id) => http.get(`/api/rooms/${id}`),

  /** 加入房间 */
  joinRoom: (id) => http.post(`/api/rooms/${id}/join`),

  /** 退出房间（owner 不可退出） */
  leaveRoom: (id) => http.post(`/api/rooms/${id}/leave`),

  /** 查询房间成员列表（含用户信息） */
  getMembers: (id) => http.get(`/api/rooms/${id}/members`),
};
