/**
 * DOM 工具：选择器与字符串安全处理，避免在业务代码中重复拼接 innerHTML。
 */

/** 在指定根节点内查询单个元素；root 缺省为 document */
export function $el(selector, root = document) {
  return root.querySelector(selector);
}

/** 在指定根节点内查询全部元素，返回数组 */
export function $els(selector, root = document) {
  return Array.from(root.querySelectorAll(selector));
}

/** 创建元素并附加到父节点 */
export function createEl(tag, className, parent) {
  const el = document.createElement(tag);
  if (className) el.className = className;
  if (parent) parent.appendChild(el);
  return el;
}

/** HTML 转义，防止 XSS（渲染用户输入时必须经过此函数） */
export function escapeHtml(text) {
  return String(text ?? '')
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#39;');
}

/** 简易防抖：连续触发时仅执行最后一次 */
export function debounce(fn, delay = 300) {
  let timer = null;
  return (...args) => {
    clearTimeout(timer);
    timer = setTimeout(() => fn(...args), delay);
  };
}
