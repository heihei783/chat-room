/**
 * Toast 轻提示组件（单例）：全局唯一的消息浮层，自动消失。
 * 页面与 services 层通过 toast.show(message, type) 统一提示。
 */
import { createEl } from '../utils/dom.js';

const TOAST_DURATION = 2200;

class Toast {
  constructor() {
    this.container = null;
    this.timer = null;
  }

  /** 确保挂载容器（#overlay）存在 */
  ensureContainer() {
    if (this.container) return;
    this.container = createEl('div', 'toast-container', document.getElementById('overlay'));
  }

  /**
   * 显示一条提示。
   * @param {string} message 文案
   * @param {'success'|'error'|'info'|'warning'} type 类型（决定样式与图标）
   */
  show(message, type = 'info') {
    this.ensureContainer();
    const el = createEl('div', `toast toast-${type}`);
    el.textContent = message;
    this.container.appendChild(el);

    // 触发入场动画后自动移除
    requestAnimationFrame(() => el.classList.add('toast-show'));
    setTimeout(() => {
      el.classList.remove('toast-show');
      setTimeout(() => el.remove(), 300);
    }, TOAST_DURATION);
  }
}

/** 单例导出 */
export const toast = new Toast();
