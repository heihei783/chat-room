/**
 * MessageInput 消息输入组件：文本输入（Enter 发送 / Shift+Enter 换行）+ 图片上传。
 * 上传走 uploadService，成功后通过 onSendImage 回调交给上层发送图片消息。
 */
import { createEl } from '../utils/dom.js';
import { uploadService } from '../services/uploadService.js';
import { toast } from './Toast.js';

/**
 * @param {object} options
 * @param {Function} options.onSendText (text) => void
 * @param {Function} options.onSendImage (url) => void
 */
export function createMessageInput({ onSendText, onSendImage } = {}) {
  const el = createEl('div', 'message-input');

  const textarea = document.createElement('textarea');
  textarea.className = 'msg-textarea';
  textarea.placeholder = '输入消息，Enter 发送，Shift+Enter 换行';
  el.appendChild(textarea);

  const toolbar = createEl('div', 'msg-toolbar', el);

  // 图片上传按钮（隐藏原生 file input）
  const fileInput = document.createElement('input');
  fileInput.type = 'file';
  fileInput.accept = 'image/*';
  fileInput.style.display = 'none';
  fileInput.addEventListener('change', () => {
    const file = fileInput.files && fileInput.files[0];
    fileInput.value = '';
    if (!file) return;
    handleUpload(file);
  });
  toolbar.appendChild(fileInput);

  const imageBtn = createEl('button', 'btn btn-sm btn-ghost', toolbar);
  imageBtn.textContent = '图片';
  imageBtn.addEventListener('click', () => fileInput.click());

  const sendBtn = createEl('button', 'btn btn-sm btn-primary', toolbar);
  sendBtn.textContent = '发送';
  sendBtn.addEventListener('click', submit);

  textarea.addEventListener('keydown', (e) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      submit();
    }
  });

  /** 上传图片 → 回调发送图片消息（含 loading 提示与错误兜底） */
  async function handleUpload(file) {
    const origin = sendBtn.textContent;
    sendBtn.disabled = true;
    sendBtn.textContent = '上传中…';
    try {
      const url = await uploadService.upload(file);
      onSendImage && (await onSendImage(url));
    } catch (err) {
      toast.show(err.message || '图片上传失败', 'error');
    } finally {
      sendBtn.disabled = false;
      sendBtn.textContent = origin;
    }
  }

  /** 提交文本消息 */
  function submit() {
    const text = textarea.value.trim();
    if (!text) return;
    onSendText && onSendText(text);
    textarea.value = '';
    textarea.focus();
  }

  /** 发送中禁用（防止连点） */
  function setSending(sending) {
    sendBtn.disabled = sending;
  }

  return { el, setSending, focus: () => textarea.focus() };
}
