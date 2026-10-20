/**
 * 文件上传业务服务：上传并返回可直接展示的完整 URL。
 */
import { uploadApi } from '../api/upload.js';
import { resolveAssetUrl } from '../config/env.js';

export const uploadService = {
  /**
   * 上传文件，返回完整可访问 URL（用于头像、图片消息）。
   * @param {File} file 浏览器文件对象
   */
  async upload(file) {
    const res = await uploadApi.upload(file);
    return resolveAssetUrl(res.data.url);
  },
};
