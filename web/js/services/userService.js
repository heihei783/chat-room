/**
 * 用户业务服务：资料查询与更新，维护 store.user。
 */
import { userApi } from '../api/user.js';
import { setState } from '../store/index.js';

export const userService = {
  /** 拉取当前登录用户资料并写入 store */
  async getProfile() {
    const res = await userApi.getProfile();
    setState({ user: res.data });
    return res.data;
  },

  /** 更新资料（nickname/avatar 至少一个），更新后同步 store 与本地用户 */
  async updateProfile(data) {
    const res = await userApi.updateProfile(data);
    setState({ user: res.data });
    return res.data;
  },
};
