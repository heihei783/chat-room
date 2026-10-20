/**
 * Avatar 头像组件：有头像显示图片，无头像显示昵称首字圆形占位。
 */
import { createEl } from '../utils/dom.js';
import { resolveAssetUrl } from '../config/env.js';

/**
 * 创建头像元素。
 * @param {object} user { id, nickname, avatar }
 * @param {number} [size=40] 直径（px）
 * @returns {HTMLElement} 头像元素
 */
export function createAvatar(user = {}, size = 40) {
  const el = createEl('div', 'avatar');
  el.style.width = `${size}px`;
  el.style.height = `${size}px`;
  el.style.fontSize = `${Math.round(size * 0.4)}px`;

  const avatarUrl = resolveAssetUrl(user.avatar);
  if (avatarUrl) {
    const img = document.createElement('img');
    img.src = avatarUrl;
    img.alt = user.nickname || 'avatar';
    img.onerror = () => {
      // 图片加载失败时回退为首字占位
      img.remove();
      el.textContent = getInitial(user.nickname);
    };
    el.appendChild(img);
  } else {
    el.textContent = getInitial(user.nickname);
  }
  return el;
}

/** 取昵称首字符作为占位文本 */
function getInitial(nickname) {
  return nickname ? nickname.trim().charAt(0).toUpperCase() : '?';
}
