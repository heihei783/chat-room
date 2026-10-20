/**
 * 前端路由：基于 location.hash 的轻量路由（零依赖）。
 * 职责：
 *   - 路由表声明（path → 视图模块）
 *   - 登录守卫：未登录仅可访问公开页，已登录自动进入聊天页
 *   - 视图渲染统一入口：清空 #app 后调用视图的 render(container)
 * 约定：视图模块导出 { title, render }；render 接收挂载容器并自行完成初始化。
 */
import { getToken } from '../api/http.js';
import { LoginView } from '../views/LoginView.js';
import { RegisterView } from '../views/RegisterView.js';
import { ForgotPasswordView } from '../views/ForgotPasswordView.js';
import { ChatView } from '../views/ChatView.js';

/** 路由表：path → 视图（title 用于浏览器标签页标题） */
const routes = {
  '/login': LoginView,
  '/register': RegisterView,
  '/forgot': ForgotPasswordView,
  '/chat': ChatView,
};

/** 公开页（无需登录） */
const PUBLIC_PATHS = ['/login', '/register', '/forgot'];

/** 解析当前 hash 对应的路径（忽略 query 部分） */
function resolvePath() {
  const hash = location.hash.replace(/^#/, '');
  return (hash.split('?')[0] || '/login').replace(/\/+$/, '') || '/login';
}

/** 渲染当前路径对应视图；守卫逻辑在此统一处理 */
function render() {
  const path = resolvePath();
  const token = getToken();

  // 未登录访问受保护页 → 强制回登录页
  if (!token && !PUBLIC_PATHS.includes(path)) {
    location.hash = '#/login';
    return;
  }
  // 已登录访问登录/注册页 → 直接进聊天页
  if (token && PUBLIC_PATHS.includes(path)) {
    location.hash = '#/chat';
    return;
  }

  const route = routes[path] || (token ? ChatView : LoginView);
  document.title = route.title ? `${route.title} · chat-app` : 'chat-app';

  const app = document.getElementById('app');
  // 切换视图前执行旧视图的清理函数（取消订阅等），避免跨页面残留监听
  if (app.__viewCleanup) {
    app.__viewCleanup();
    app.__viewCleanup = null;
  }
  app.innerHTML = '';
  route.render(app);
}

/** 编程式跳转 */
export function navigate(path) {
  location.hash = path;
}

/** 初始化路由：注册 hashchange 并执行首次渲染 */
export function initRouter() {
  window.addEventListener('hashchange', render);
  render();
}
