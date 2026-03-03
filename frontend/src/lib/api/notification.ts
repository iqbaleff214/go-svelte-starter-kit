import { api } from './client';
import type { NotificationListResponse, UnreadCountResponse } from '$types';

export const notificationApi = {
	list(page = 1, limit = 20): Promise<NotificationListResponse> {
		return api.get(`/notifications?page=${page}&limit=${limit}`);
	},

	unreadCount(): Promise<UnreadCountResponse> {
		return api.get('/notifications/unread-count');
	},

	markRead(id: string): Promise<void> {
		return api.patch(`/notifications/${id}/read`);
	},

	markAllRead(): Promise<void> {
		return api.patch('/notifications/read-all');
	},

	sendTest(): Promise<void> {
		return api.post('/notifications/test');
	}
};
