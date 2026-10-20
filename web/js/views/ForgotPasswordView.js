/**
 * ForgotPasswordView 忘记密码页：邮箱 + 验证码 + 新密码，重置成功后跳登录。
 */
import { $el } from '../utils/dom.js';
import { validateEmail, validatePassword, validateCode } from '../utils/validate.js';
import { authService } from '../services/authService.js';
import { toast } from '../components/Toast.js';

export const ForgotPasswordView = {
  title: '重置密码',

  render(container) {
    container.innerHTML = `
      <div class="auth-page">
        <div class="auth-card">
          <div class="auth-brand">
            <img src="./assets/logo.svg" alt="logo">
            <span class="auth-title">重置密码</span>
          </div>
          <p class="auth-subtitle">通过邮箱验证码重置密码</p>

          <form id="forgot-form">
            <div class="form-group">
              <label for="fp-email">邮箱</label>
              <input class="form-input" id="fp-email" type="email" placeholder="注册时使用的邮箱" autocomplete="email">
            </div>
            <div class="form-group">
              <label for="fp-code">邮箱验证码</label>
              <div class="code-row">
                <input class="form-input" id="fp-code" placeholder="6 位数字" inputmode="numeric">
                <button class="btn btn-ghost code-btn" id="fp-code-btn" type="button">获取验证码</button>
              </div>
            </div>
            <div class="form-group">
              <label for="fp-password">新密码</label>
              <input class="form-input" id="fp-password" type="password" placeholder="至少 6 位" autocomplete="new-password">
            </div>
            <button class="btn btn-primary btn-block" id="forgot-submit" type="submit">重置密码</button>
          </form>

          <div class="auth-links">
            <a href="#/login">想起密码了，去登录</a>
          </div>
        </div>
      </div>
    `;

    const form = $el('#forgot-form', container);
    const submitBtn = $el('#forgot-submit', container);
    const codeBtn = $el('#fp-code-btn', container);

    /** 发送重置验证码 + 60s 倒计时 */
    let countdownTimer = null;
    async function sendCode() {
      const email = $el('#fp-email', container).value.trim();
      const v = validateEmail(email);
      if (!v.valid) return toast.show(v.message, 'error');
      codeBtn.disabled = true;
      try {
        await authService.sendResetCode(email);
        toast.show('验证码已发送到邮箱', 'success');
        let seconds = 60;
        codeBtn.textContent = `${seconds}s`;
        countdownTimer = setInterval(() => {
          seconds -= 1;
          if (seconds <= 0) {
            clearInterval(countdownTimer);
            codeBtn.disabled = false;
            codeBtn.textContent = '获取验证码';
          } else {
            codeBtn.textContent = `${seconds}s`;
          }
        }, 1000);
      } catch (err) {
        toast.show(err.message || '发送失败', 'error');
        codeBtn.disabled = false;
      }
    }
    codeBtn.addEventListener('click', sendCode);

    /** 提交重置 */
    form.addEventListener('submit', async (e) => {
      e.preventDefault();
      const email = $el('#fp-email', container).value.trim();
      const code = $el('#fp-code', container).value.trim();
      const password = $el('#fp-password', container).value;

      const vEmail = validateEmail(email);
      if (!vEmail.valid) return toast.show(vEmail.message, 'error');
      const vCode = validateCode(code);
      if (!vCode.valid) return toast.show(vCode.message, 'error');
      const vPwd = validatePassword(password);
      if (!vPwd.valid) return toast.show(vPwd.message, 'error');

      submitBtn.disabled = true;
      try {
        await authService.resetPassword({ email, code, new_password: password });
        toast.show('密码已重置，请登录', 'success');
        location.hash = '#/login';
      } catch (err) {
        toast.show(err.message || '重置失败', 'error');
      } finally {
        submitBtn.disabled = false;
      }
    });
  },
};
