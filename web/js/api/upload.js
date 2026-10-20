/**
 * 文件上传接口（需 JWT）：返回可访问 URL，用于头像、图片消息等。
 */
import { http } from './http.js';

export const uploadApi = {
  /** 上传文件（multipart），返回 { url, name, size } */
  upload: (file) => http.upload('/api/upload', file),
};
