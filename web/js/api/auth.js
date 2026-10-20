/**
 * 认证模块接口：验证码、注册、登录、密码重置（均为公开接口）。
 * 本模块只负责请求，业务编排（token 持久化等）在 services/authService.js。
 */
import { http } from './http.js';

export const authApi = {
  /** 发送注册邮箱验证码 */
  sendEmailCode: (email) => http.post('/api/auth/email-code', { email }),

  /** 发送密码重置验证码（不校验邮箱是否注册，防枚举） */
  sendResetCode: (email) => http.post('/api/auth/password/reset-code', { email }),

  /** 注册：username/email/password/code，nickname 可选 */
  register: (data) => http.post('/api/auth/register', data),

  /** 登录：email/password，成功返回 { token, user } */
  login: (data) => http.post('/api/auth/login', data),

  /** 重置密码：email/code/new_password */
  resetPassword: (data) => http.post('/api/auth/password/reset', data),
};
