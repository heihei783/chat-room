/**
 * 房间业务服务：创建/查询/加入/退出/成员，维护 store.rooms 与 store.roomMembers。
 */
import { roomApi } from '../api/room.js';
import { getState, setState } from '../store/index.js';
import { wsClient } from '../websocket/index.js';

export const roomService = {
  /** 创建房间，成功后刷新房间列表 */
  async createRoom(data) {
    const res = await roomApi.createRoom(data);
    await this.getRooms();
    return res.data;
  },

  /** 拉取当前用户房间列表 */
  async getRooms() {
    const res = await roomApi.getRooms();
    setState({ rooms: res.data || [] });
    return res.data || [];
  },

  /** 查询房间详情 */
  async getRoom(id) {
    const res = await roomApi.getRoom(id);
    return res.data;
  },

  /** 加入房间：更新列表并通过 WebSocket 订阅该房间的群聊推送 */
  async joinRoom(id) {
    await roomApi.joinRoom(id);
    wsClient.joinRoom(id);
    await this.getRooms();
  },

  /** 退出房间：更新列表并取消 WebSocket 订阅 */
  async leaveRoom(id) {
    await roomApi.leaveRoom(id);
    wsClient.leaveRoom(id);
    // 若退出的正是当前会话，清空会话
    const { activeRoom } = getState();
    if (activeRoom && activeRoom.id === id) {
      setState({ activeRoom: null, messages: [] });
    }
    await this.getRooms();
  },

  /** 拉取房间成员列表（含用户信息与角色） */
  async getMembers(roomId) {
    const res = await roomApi.getMembers(roomId);
    setState({ roomMembers: res.data || [] });
    return res.data || [];
  },
};
