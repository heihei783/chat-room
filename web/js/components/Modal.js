/**
 * Modal 弹窗组件（函数式）：打开确认弹窗，返回关闭函数。
 * 表单类内容由调用方构造 body 传入，保持组件通用。
 */
import { createEl } from '../utils/dom.js';

/**
 * 打开一个弹窗。
 * @param {object} options
 * @param {string} options.title 标题
 * @param {HTMLElement|string} options.body 内容（元素或 HTML 字符串；字符串会被插入 innerHTML，调用方需自行转义）
 * @param {string} [options.okText='确定'] 确认按钮文案
 * @param {boolean} [options.showOk=true] 是否显示确认按钮（纯展示场景可关闭）
 * @param {Function} [options.onOk] 点击确认回调（返回 false / reject 时不关闭弹窗）
 * @returns {Function} close 关闭弹窗
 */
export function openModal({ title = '', body = '', okText = '确定', showOk = true, onOk } = {}) {
  const overlay = document.getElementById('overlay');
  const mask = createEl('div', 'modal-mask');
  const modal = createEl('div', 'modal');

  const header = createEl('div', 'modal-header');
  header.innerHTML = `<h3>${title}</h3>`;
  const closeBtn = createEl('button', 'modal-close', header);
  closeBtn.innerHTML = '&times;';

  const content = createEl('div', 'modal-body', modal);
  if (typeof body === 'string') content.innerHTML = body;
  else if (body) content.appendChild(body);

  const footer = createEl('div', 'modal-footer', modal);
  const cancelBtn = createEl('button', 'btn btn-ghost', footer);
  cancelBtn.textContent = '取消';
  if (showOk) {
    const okBtn = createEl('button', 'btn btn-primary', footer);
    okBtn.textContent = okText;
    okBtn.addEventListener('click', async () => {
      const result = onOk ? await onOk() : true;
      if (result !== false) close();
    });
  }

  mask.appendChild(modal);
  overlay.appendChild(mask);

  function close() {
    mask.remove();
  }

  closeBtn.addEventListener('click', close);
  cancelBtn.addEventListener('click', close);
  mask.addEventListener('click', (e) => {
    if (e.target === mask) close();
  });

  return close;
}

/** 便捷：确认对话框 */
export function confirmDialog(title, message, onOk, okText = '确定') {
  return openModal({
    title,
    body: `<p class="text-secondary">${message}</p>`,
    okText,
    onOk,
  });
}
