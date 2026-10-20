/**
 * 校验工具：表单校验规则集中定义，返回 { valid, message }，页面可直接用于提示。
 */

/** 邮箱格式校验 */
export function isEmail(value) {
  return /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(value);
}

/** 校验邮箱，非法时返回提示信息 */
export function validateEmail(email) {
  if (!email) return { valid: false, message: '请输入邮箱' };
  if (!isEmail(email)) return { valid: false, message: '邮箱格式不正确' };
  return { valid: true, message: '' };
}

/** 校验密码（至少6位） */
export function validatePassword(password) {
  if (!password) return { valid: false, message: '请输入密码' };
  if (password.length < 6) return { valid: false, message: '密码至少6位' };
  return { valid: true, message: '' };
}

/** 校验验证码（6位数字） */
export function validateCode(code) {
  if (!code) return { valid: false, message: '请输入验证码' };
  if (!/^\d{6}$/.test(code)) return { valid: false, message: '验证码为6位数字' };
  return { valid: true, message: '' };
}

/** 校验必填项 */
export function validateRequired(value, label = '内容') {
  if (!value || !String(value).trim()) return { valid: false, message: `请输入${label}` };
  return { valid: true, message: '' };
}

/** 校验用户名（1-20位，不含空白字符） */
export function validateUsername(username) {
  if (!username) return { valid: false, message: '请输入用户名' };
  if (/\s/.test(username)) return { valid: false, message: '用户名不能包含空格' };
  if (username.length > 20) return { valid: false, message: '用户名不能超过20个字符' };
  return { valid: true, message: '' };
}
