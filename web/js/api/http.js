/**
 * HTTP 请求核心封装：所有接口调用的唯一出口。
 * 职责：
 *   1. 统一拼接 baseURL 与 query 参数
 *   2. 自动注入 JWT（Authorization: Bearer）
 *   3. 统一 JSON 解析与错误处理（业务失败抛 Error，message 可直接展示）
 *   4. 401 时清除本地登录态
 * 约定：后端响应结构 { success, message, data }，HTTP 非 2xx 或 success=false 均视为失败。
 */
import { ENV } from '../config/env.js';
import * as storage from '../utils/storage.js';

const TOKEN_KEY = 'token';

/** 读取本地 JWT（供 http 封装与 WebSocket 连接共用） */
export function getToken() {
  return storage.get(TOKEN_KEY) || '';
}

/** 保存 JWT（登录成功后调用） */
export function setToken(token) {
  storage.set(TOKEN_KEY, token);
}

/** 清除 JWT（登出或 401 时调用） */
export function clearToken() {
  storage.remove(TOKEN_KEY);
}

/** 拼接 query 字符串；跳过空值 */
function buildQuery(params) {
  if (!params) return '';
  const usp = new URLSearchParams();
  Object.entries(params).forEach(([k, v]) => {
    if (v !== undefined && v !== null && v !== '') usp.append(k, v);
  });
  const qs = usp.toString();
  return qs ? `?${qs}` : '';
}

/**
 * 核心请求函数。
 * @param {string} path 接口路径（如 /api/auth/login）
 * @param {object} options { method, body, formData, auth }
 * @returns {Promise<object>} 后端响应对象（含 success/data）
 * @throws {Error} 失败时抛出，message 为可展示的错误信息
 */
async function request(path, options = {}) {
  const { method = 'GET', body, formData, auth = true } = options;

  const headers = {};
  if (body && !formData) headers['Content-Type'] = 'application/json';
  if (auth) {
    const token = getToken();
    if (token) headers['Authorization'] = `Bearer ${token}`;
  }

  const fetchOptions = { method, headers };
  if (body) fetchOptions.body = JSON.stringify(body);
  if (formData) fetchOptions.body = formData; // 不手动设 Content-Type，浏览器自动带 boundary

  let res;
  try {
    res = await fetch(`${ENV.API_BASE}${path}`, fetchOptions);
  } catch {
    throw new Error('网络请求失败，请确认服务已启动');
  }

  // 尝试解析 JSON；非 JSON（如纯文本错误页）时保留原文
  const text = await res.text();
  let data = null;
  if (text) {
    try {
      data = JSON.parse(text);
    } catch {
      data = null;
    }
  }

  // 401：token 失效，清除登录态（页面收到错误后由路由守卫跳转登录页）
  if (res.status === 401 && auth) {
    clearToken();
  }

  if (!res.ok || (data && data.success === false)) {
    const message = (data && data.message) || `请求失败（HTTP ${res.status}）`;
    const err = new Error(message);
    err.status = res.status;
    throw err;
  }

  return data;
}

/** 领域接口层统一导出的 http 工具 */
export const http = {
  /** GET 请求；params 以 query 形式拼接 */
  get: (path, params, options) => request(`${path}${buildQuery(params)}`, { method: 'GET', ...options }),

  /** POST 请求 */
  post: (path, body, options) => request(path, { method: 'POST', body, ...options }),

  /** PUT 请求 */
  put: (path, body, options) => request(path, { method: 'PUT', body, ...options }),

  /** 上传文件：multipart/form-data */
  upload: (path, file, options) => {
    const formData = new FormData();
    formData.append('file', file);
    return request(path, { method: 'POST', formData, ...options });
  },
};
