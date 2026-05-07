import { ref, onUnmounted } from 'vue';
import { addWsHandler, sendMessage } from './useWebSocket';
export interface UserStatusInfo {
 userId: number;
 username: string;
 nickname: string;
 avatar: string;
 status: 'online' | 'offline' | 'busy';
 lastOnline: number;
 timestamp: number;
}
const userStatusMap = ref<Map<number, UserStatusInfo>>(new Map());
const subscribedUsers = ref<Set<number>>(new Set());
export function useUserStatus() {
 const subscribeUserStatus = (userId: number) => {
 if (subscribedUsers.value.has(userId)) {
 return;
 }
 subscribedUsers.value.add(userId);
 sendMessage({
 type: 'subscribe_user_status',
 data: { user_id: userId }
 });
 };
 const unsubscribeUserStatus = (userId: number) => {
 subscribedUsers.value.delete(userId);
 sendMessage({
 type: 'unsubscribe_user_status',
 data: { user_id: userId }
 });
 };
 const getUserStatus = (userId: number): UserStatusInfo | undefined => {
 return userStatusMap.value.get(userId);
 };
 const isUserOnline = (userId: number): boolean => {
 const status = userStatusMap.value.get(userId);
 return status?.status === 'online';
 };
 const formatLastOnline = (lastOnline: number): string => {
 if (!lastOnline)
 return '';
 const now = Date.now() / 1000;
 const diff = now - lastOnline;
 if (diff < 60) {
 return '刚刚在线';
 }
 else if (diff < 3600) {
 return `${Math.floor(diff / 60)}分钟前`;
 }
 else if (diff < 86400) {
 return `${Math.floor(diff / 3600)}小时前`;
 }
 else if (diff < 604800) {
 return `${Math.floor(diff / 86400)}天前`;
 }
 else {
 const date = new Date(lastOnline * 1000);
 return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')}`;
 }
 };
 const handleStatusChange = (data: UserStatusInfo) => {
 userStatusMap.value.set(data.userId, data);
 };
 const cleanup = addWsHandler((message) => {
 if (message.type === 'user_status_changed') {
 handleStatusChange(message.data);
 }
 }, 'user_status_changed');
 onUnmounted(() => {
 cleanup();
 });
 return {
 userStatusMap,
 subscribedUsers,
 subscribeUserStatus,
 unsubscribeUserStatus,
 getUserStatus,
 isUserOnline,
 formatLastOnline
 };
}