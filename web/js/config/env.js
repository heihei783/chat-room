/**
 * 环境配置：集中管理后端服务地址，避免散落各处。
 * 部署形态说明：
 *   1. 同源部署（推荐，前端由后端托管）：自动使用当前页面地址（location），
 *      后端改端口后前端无需任何改动
 *   2. 独立部署（前端静态服务与后端不同源）：在 index.html 内联
 *      window.__APP_ENV__ = { API_BASE, WS_BASE } 覆盖
 */
const userConfig = (typeof window !== 'undefined' && window.__APP_ENV__) || {};

// 同源部署：API 与 WebSocket 自动跟随当前页面协议与主机（含端口）
const sameOrigin = {
  API_BASE: window.location.origin,
  WS_BASE: `${window.location.protocol === 'https:' ? 'wss:' : 'ws:'}//${window.location.host}`,
};

export const ENV = {
  API_BASE: userConfig.API_BASE || sameOrigin.API_BASE,
  WS_BASE: userConfig.WS_BASE || sameOrigin.WS_BASE,
};

/** 拼接静态资源完整地址（上传接口返回的是 /uploads/... 相对路径） */
export function resolveAssetUrl(url) {
  if (!url) return '';
  if (/^https?:\/\//.test(url) || url.startsWith('data:')) return url;
  return `${ENV.API_BASE}${url}`;
}
