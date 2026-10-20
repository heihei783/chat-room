/**
 * RegisterView 注册页：用户名 + 邮箱 + 验证码 + 密码（昵称可选）。
 * 注册成功后引导跳转登录页。
 */
import { $el } from '../utils/dom.js';
import { validateEmail, validatePassword, validateCode, validateUsername } from '../utils/validate.js';
import { authService } from '../services/authService.js';
import { toast } from '../components/Toast.js';

export const RegisterView = {
  title: '注册',

  render(container) {
    container.innerHTML = `
      <div class="auth-page">
        <div class="auth-card">
          <div class="auth-brand">
            <img src="./assets/logo.svg" alt="logo">
            <span class="auth-title">注册账号</span>
          </div>
          <p class="auth-subtitle">使用邮箱验证码注册</p>

          <form id="register-form">
            <div class="form-group">
              <label for="reg-username">用户名</label>
              <input class="form-input" id="reg-username" placeholder="1-20 个字符" autocomplete="username">
            </div>
            <div class="form-group">
              <label for="reg-nickname">昵称（选填）</label>
              <input class="form-input" id="reg-nickname" placeholder="聊天中显示的名字">
            </div>
            <div class="form-group">
              <label for="reg-email">邮箱</label>
              <input class="form-input" id="reg-email" type="email" placeholder="用于登录与接收验证码" autocomplete="email">
            </div>
            <div class="form-group">
              <label for="reg-code">邮箱验证码</label>
              <div class="code-row">
                <input class="form-input" id="reg-code" placeholder="6 位数字" inputmode="numeric">
                <button class="btn btn-ghost code-btn" id="reg-code-btn" type="button">获取验证码</button>
              </div>
            </div>
            <div class="form-group">
              <label for="reg-password">密码</label>
              <input class="form-input" id="reg-password" type="password" placeholder="至少 6 位" autocomplete="new-password">
            </div>
            <button class="btn btn-primary btn-block" id="register-submit" type="submit">注 册</button>
          </form>

          <div class="auth-links">
            <a href="#/login">已有账号，去登录</a>
          </div>
        </div>
      </div>
    `;

    const form = $el('#register-form', container);
    const submitBtn = $el('#register-submit', container);
    const codeBtn = $el('#reg-code-btn', container);

    /** 发送验证码 + 60s 倒计时 */
    let countdownTimer = null;
    async function sendCode() {
      const email = $el('#reg-email', container).value.trim();
      const v = validateEmail(email);
      if (!v.valid) return toast.show(v.message, 'error');
      codeBtn.disabled = true;
      try {
        await authService.sendEmailCode(email);
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

    /** 注册提交 */
    form.addEventListener('submit', async (e) => {
      e.preventDefault();
      const username = $el('#reg-username', container).value.trim();
      const nickname = $el('#reg-nickname', container).value.trim();
      const email = $el('#reg-email', container).value.trim();
      const code = $el('#reg-code', container).value.trim();
      const password = $el('#reg-password', container).value;

      const vUser = validateUsername(username);
      if (!vUser.valid) return toast.show(vUser.message, 'error');
      const vEmail = validateEmail(email);
      if (!vEmail.valid) return toast.show(vEmail.message, 'error');
      const vCode = validateCode(code);
      if (!vCode.valid) return toast.show(vCode.message, 'error');
      const vPwd = validatePassword(password);
      if (!vPwd.valid) return toast.show(vPwd.message, 'error');

      submitBtn.disabled = true;
      try {
        await authService.register({ username, email, password, code, nickname });
        toast.show('注册成功，请登录', 'success');
        location.hash = '#/login';
      } catch (err) {
        toast.show(err.message || '注册失败', 'error');
      } finally {
        submitBtn.disabled = false;
      }
    });
  },
};
