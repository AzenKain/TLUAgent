import React, { useState, useRef, useEffect } from 'react';
import { useTranslation } from 'react-i18next';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Bell, CheckCheck, FileText, AlertCircle, Info } from 'lucide-react';
import { useNavigate } from 'react-router-dom';
import { inquiryService, type NotificationDTO } from '@/services/inquiryService';

const getNotificationIcon = (type: string) => {
  switch (type) {
    case 'SUPERSEDED_PROPOSED':
      return <AlertCircle className="h-4 w-4 text-amber-400" />;
    case 'INQUIRY_ANSWERED':
      return <FileText className="h-4 w-4 text-tlu-blue" />;
    default:
      return <Info className="h-4 w-4 text-slate-400" />;
  }
};

export const NotificationBell: React.FC = () => {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const [isOpen, setIsOpen] = useState(false);
  const dropdownRef = useRef<HTMLDivElement>(null);

  const { data: unreadCount = 0 } = useQuery({
    queryKey: ['notifications', 'unread-count'],
    queryFn: () => inquiryService.getUnreadCount(),
    staleTime: 60000,
    refetchInterval: 120000,
    refetchIntervalInBackground: false,
    refetchOnWindowFocus: true,
  });

  const { data: notifData } = useQuery({
    queryKey: ['notifications', 'list'],
    queryFn: () => inquiryService.getNotifications(1, 10),
    enabled: isOpen,
  });

  const markReadMutation = useMutation({
    mutationFn: (id: string) => inquiryService.markNotificationAsRead(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['notifications'] });
    },
  });

  const markAllReadMutation = useMutation({
    mutationFn: () => inquiryService.markAllNotificationsAsRead(),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['notifications'] });
    },
  });

  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (dropdownRef.current && !dropdownRef.current.contains(event.target as Node)) {
        setIsOpen(false);
      }
    };
    if (isOpen) {
      document.addEventListener('mousedown', handleClickOutside);
    }
    return () => {
      document.removeEventListener('mousedown', handleClickOutside);
    };
  }, [isOpen]);

  const handleNotificationClick = (notif: NotificationDTO) => {
    if (!notif.is_read) {
      markReadMutation.mutate(notif.id);
    }
    setIsOpen(false);
    if (notif.inquiry_id) {
      navigate(`/admin/inquiries?id=${notif.inquiry_id}`);
    }
  };

  return (
    <div className="relative" ref={dropdownRef}>
      <button
        type="button"
        onClick={() => setIsOpen(!isOpen)}
        className="relative flex h-11 w-11 shrink-0 items-center justify-center rounded-lg text-slate-500 transition hover:bg-slate-100 hover:text-slate-700 dark:text-slate-400 dark:hover:bg-slate-800 dark:hover:text-slate-200"
        aria-label={t('notifications.title')}
      >
        <Bell className="h-5 w-5" />
        {unreadCount > 0 && (
          <span className="absolute right-2 top-2 flex h-4 min-w-4 items-center justify-center rounded-full bg-tlu-red px-1 text-[10px] font-bold text-white shadow-sm ring-2 ring-white dark:ring-slate-900">
            {unreadCount > 99 ? '99+' : unreadCount}
          </span>
        )}
      </button>

      {isOpen && (
        <div className="absolute right-0 mt-2 w-80 sm:w-96 rounded-2xl border border-slate-200 bg-white shadow-2xl backdrop-blur-md dark:border-blue-900/40 dark:bg-[#111A35]/95 z-50 overflow-hidden">
          <div className="flex items-center justify-between border-b border-slate-100 px-4 py-3 dark:border-blue-900/30">
            <div className="flex items-center gap-2">
              <span className="text-sm font-semibold text-slate-800 dark:text-slate-100">
                {t('notifications.title')}
              </span>
              {unreadCount > 0 && (
                <span className="rounded-full bg-blue-100 px-2 py-0.5 text-xs font-semibold text-tlu-navy dark:bg-blue-950/60 dark:text-blue-300">
                  {unreadCount}
                </span>
              )}
            </div>
            {unreadCount > 0 && (
              <button
                type="button"
                onClick={() => markAllReadMutation.mutate()}
                className="flex items-center gap-1 text-xs font-medium text-tlu-blue transition hover:text-tlu-navy dark:text-blue-400 dark:hover:text-blue-300"
              >
                <CheckCheck className="h-3.5 w-3.5" />
                <span>{t('notifications.mark_all_read')}</span>
              </button>
            )}
          </div>

          <div className="max-h-80 overflow-y-auto divide-y divide-slate-100 dark:divide-blue-900/30">
            {!notifData?.items || notifData.items.length === 0 ? (
              <div className="flex h-32 flex-col items-center justify-center text-slate-400">
                <Bell className="h-6 w-6 stroke-[1.5] text-slate-300 dark:text-slate-600" />
                <p className="mt-2 text-xs">{t('notifications.empty')}</p>
              </div>
            ) : (
              notifData.items.map((item) => (
                <div
                  key={item.id}
                  onClick={() => handleNotificationClick(item)}
                  className={`flex cursor-pointer gap-3 px-4 py-3 transition hover:bg-slate-50 dark:hover:bg-blue-950/20 ${
                    !item.is_read ? 'bg-blue-50/40 dark:bg-blue-950/30' : ''
                  }`}
                >
                  <div className="mt-0.5 shrink-0">{getNotificationIcon(item.type)}</div>
                  <div className="min-w-0 flex-1">
                    <p className={`text-xs ${!item.is_read ? 'font-semibold text-slate-900 dark:text-slate-100' : 'text-slate-700 dark:text-slate-300'}`}>
                      {item.title}
                    </p>
                    <p className="mt-1 line-clamp-2 text-xs text-slate-500 dark:text-slate-400">
                      {item.content}
                    </p>
                    <span className="mt-1.5 block text-[10px] text-slate-400">
                      {new Date(item.created_at).toLocaleString()}
                    </span>
                  </div>
                  {!item.is_read && (
                    <span className="mt-1.5 h-2 w-2 shrink-0 rounded-full bg-tlu-blue" />
                  )}
                </div>
              ))
            )}
          </div>
        </div>
      )}
    </div>
  );
};
