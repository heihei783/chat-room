/**
 * 格式化工具：时间、字节数等展示层通用函数，与业务无关。
 */

/** 将 ISO 时间格式化为友好显示：今天/昨天 HH:mm，更早显示 M月D日 */
export function formatTime(iso) {
  if (!iso) return '';
  const date = new Date(iso);
  if (Number.isNaN(date.getTime())) return '';

  const now = new Date();
  const startOfToday = new Date(now.getFullYear(), now.getMonth(), now.getDate());
  const target = new Date(date.getFullYear(), date.getMonth(), date.getDate());
  const diffDays = Math.round((startOfToday - target) / 86400000);
  const hhmm = `${pad(date.getHours())}:${pad(date.getMinutes())}`;

  if (diffDays === 0) return `今天 ${hhmm}`;
  if (diffDays === 1) return `昨天 ${hhmm}`;
  return `${date.getMonth() + 1}月${date.getDate()}日 ${hhmm}`;
}

function pad(n) {
  return n < 10 ? `0${n}` : String(n);
}

/** 字节数转可读大小（B/KB/MB/GB） */
export function formatSize(bytes) {
  if (!bytes && bytes !== 0) return '';
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 ** 2) return `${(bytes / 1024).toFixed(1)} KB`;
  if (bytes < 1024 ** 3) return `${(bytes / 1024 ** 2).toFixed(1)} MB`;
  return `${(bytes / 1024 ** 3).toFixed(1)} GB`;
}

/** 截断长文本，超出部分以省略号结尾 */
export function truncate(text, max = 30) {
  const str = String(text || '');
  return str.length > max ? `${str.slice(0, max)}…` : str;
}
