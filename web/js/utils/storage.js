/**
 * 本地存储封装：统一 JSON 序列化与异常处理，避免业务层直接操作 localStorage。
 * 所有 key 带统一前缀，防止与其他应用冲突。
 */

const PREFIX = 'chat_app_';

/** 读取并反序列化 localStorage 值；缺失或损坏时返回 null */
export function get(key) {
  try {
    const raw = localStorage.getItem(PREFIX + key);
    return raw ? JSON.parse(raw) : null;
  } catch {
    return null;
  }
}

/** 序列化后写入 localStorage */
export function set(key, value) {
  try {
    localStorage.setItem(PREFIX + key, JSON.stringify(value));
  } catch {
    /* 存储满或隐私模式下的写入失败，忽略（不影响主流程） */
  }
}

/** 移除指定 key */
export function remove(key) {
  localStorage.removeItem(PREFIX + key);
}
