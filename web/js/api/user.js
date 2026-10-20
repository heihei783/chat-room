/**
 * 用户模块接口：当前登录用户资料（均需 JWT）。
 */
import { http } from './http.js';

export const userApi = {
  /** 获取当前登录用户资料 */
  getProfile: () => http.get('/api/user/profile'),

  /** 更新当前登录用户资料（nickname/avatar 至少一个） */
  updateProfile: (data) => http.put('/api/user/profile', data),
};
