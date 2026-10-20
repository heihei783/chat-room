/**
 * 认证业务服务：注册/登录/登出/验证码/重置密码。
 * 负责 token 持久化与登录态写入 store，是认证模块对页面的唯一入口。
 */
import { authApi } from '../api/auth.js';
import { setToken, clearToken } from '../api/http.js';
import { setState, resetAuth } from '../store/index.js';
import { closeSocket } from '../websocket/index.js';

export const authService = {
  /** 发送注册验证码 */
  async sendEmailCode(email) {
    return authApi.sendEmailCode(email);
  },

  /** 发送密码重置验证码 */
  async sendResetCode(email) {
    return authApi.sendResetCode(email);
  },

  /** 注册（成功后由页面引导跳转登录） */
  async register(data) {
    return authApi.register(data);
  },

  /** 登录：保存 token 与用户信息到 store */
  async login(email, password) {
    const res = await authApi.login({ email, password });
    setToken(res.data.token);
    setState({ token: res.data.token, user: res.data.user });
    return res.data;
  },

  /** 重置密码 */
  async resetPassword(data) {
    return authApi.resetPassword(data);
  },

  /** 登出：断开 WebSocket、清除本地登录态并重置全局状态 */
  logout() {
    closeSocket();
    clearToken();
    resetAuth();
  },
};
