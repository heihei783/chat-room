/**
 * 应用入口：初始化路由、恢复登录态、建立 WebSocket 长连接。
 * 职责仅限启动装配；各模块的初始化细节由其自身负责。
 */
import { initRouter } from './router/index.js';
import { getToken } from './api/http.js';
import { userService } from './services/userService.js';
import { wsClient } from './websocket/index.js';

// 启动路由（内部完成首次渲染与登录守卫）
initRouter();

// 恢复登录态：本地有 token 时拉取最新资料并连接 WebSocket
if (getToken()) {
  userService.getProfile().catch(() => {
    /* 资料拉取失败（如 token 过期）由路由守卫在下次导航时兜底 */
  });
  wsClient.connect();
}
