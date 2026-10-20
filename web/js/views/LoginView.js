/**
 * LoginView 登录页：邮箱 + 密码登录。
 * 成功登录后由路由守卫自动跳转聊天页（#/chat）。
 */
import { $el } from '../utils/dom.js';
import { validateEmail, validatePassword } from '../utils/validate.js';
import { authService } from '../services/authService.js';
import { toast } from '../components/Toast.js';

export const LoginView = {
  title: '登录',

  render(container) {
    container.innerHTML = `
      <div class="auth-page">
        <div class="auth-card">
          <div class="auth-brand">
            <img src="./assets/logo.svg" alt="logo">
            <span class="auth-title">chat-app</span>
          </div>
          <p class="auth-subtitle">登录后开始聊天</p>

          <form id="login-form">
            <div class="form-group">
              <label for="login-email">邮箱</label>
              <input class="form-input" id="login-email" type="email" placeholder="请输入邮箱" autocomplete="email">
            </div>
            <div class="form-group">
              <label for="login-password">密码</label>
              <input class="form-input" id="login-password" type="password" placeholder="请输入密码" autocomplete="current-password">
            </div>
            <button class="btn btn-primary btn-block" id="login-submit" type="submit">登 录</button>
          </form>

          <div class="auth-links">
            <a href="#/register">注册账号</a>
            <a href="#/forgot">忘记密码？</a>
          </div>
          <p class="auth-footer">未注册账号也可直接注册</p>
        </div>
      </div>
    `;

    const form = $el('#login-form', container);
    const submitBtn = $el('#login-submit', container);

    /** 登录提交：校验 → 调用服务 → 跳转 */
    form.addEventListener('submit', async (e) => {
      e.preventDefault();
      const email = $el('#login-email', container).value.trim();
      const password = $el('#login-password', container).value;

      const vEmail = validateEmail(email);
      if (!vEmail.valid) return toast.show(vEmail.message, 'error');
      const vPwd = validatePassword(password);
      if (!vPwd.valid) return toast.show(vPwd.message, 'error');

      submitBtn.disabled = true;
      try {
        await authService.login(email, password);
        toast.show('登录成功', 'success');
        location.hash = '#/chat';
      } catch (err) {
        toast.show(err.message || '登录失败', 'error');
      } finally {
        submitBtn.disabled = false;
      }
    });
  },
};
