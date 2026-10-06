import { lazy, Suspense } from 'react';
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import { QueryClientProvider } from '@tanstack/react-query';
import { queryClient } from '@/lib/queryClient';

import { AdminLayout } from '@/components/layout/AdminLayout';
import { ProtectedRoute } from '@/components/auth/ProtectedRoute';
import { LoadingSpinner } from '@/components/common/LoadingSpinner';

const LoginPage = lazy(() => import('@/pages/auth/LoginPage').then((m) => ({ default: m.LoginPage })));
const SetupPage = lazy(() => import('@/pages/auth/SetupPage').then((m) => ({ default: m.SetupPage })));
const DashboardPage = lazy(() => import('@/pages/dashboard/DashboardPage').then((m) => ({ default: m.DashboardPage })));
const UsersListPage = lazy(() => import('@/pages/users/UsersListPage').then((m) => ({ default: m.UsersListPage })));
const RolesListPage = lazy(() => import('@/pages/roles/RolesListPage').then((m) => ({ default: m.RolesListPage })));
const LLMManagePage = lazy(() => import('@/pages/admin/llm/LLMManagePage').then((m) => ({ default: m.LLMManagePage })));
const ChatManagePage = lazy(() => import('@/pages/admin/chat/ChatManagePage').then((m) => ({ default: m.ChatManagePage })));
const DocumentManagePage = lazy(() => import('@/pages/admin/documents/DocumentManagePage').then((m) => ({ default: m.DocumentManagePage })));
const KnowledgeGraphPage = lazy(() => import('@/pages/admin/knowledge-graph/KnowledgeGraphPage').then((m) => ({ default: m.KnowledgeGraphPage })));
const AgentManagePage = lazy(() => import('@/pages/admin/agent/AgentManagePage').then((m) => ({ default: m.AgentManagePage })));
const JobManagePage = lazy(() => import('@/pages/admin/jobs/JobManagePage').then((m) => ({ default: m.JobManagePage })));
const InquiryManagePage = lazy(() => import('@/pages/admin/inquiries/InquiryManagePage').then((m) => ({ default: m.InquiryManagePage })));
const ProfilePage = lazy(() => import('@/pages/profile/ProfilePage').then((m) => ({ default: m.ProfilePage })));
const ChatAdvisoryPage = lazy(() => import('@/pages/chat/ChatAdvisoryPage').then((m) => ({ default: m.ChatAdvisoryPage })));
const NotFoundPage = lazy(() => import('@/pages/NotFoundPage').then((m) => ({ default: m.NotFoundPage })));

export function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <BrowserRouter>
        <Suspense
          fallback={
            <div className="flex h-screen w-full items-center justify-center bg-slate-900">
              <LoadingSpinner size="lg" />
            </div>
          }
        >
        <Routes>
          {/* Public Chat Routes: Guest and Authenticated users */}
          <Route path="/" element={<ChatAdvisoryPage />} />
          <Route path="/chat" element={<Navigate to="/" replace />} />

          {/* Auth Routes */}
          <Route path="/login" element={<LoginPage />} />
          <Route path="/setup" element={<SetupPage />} />

          {/* Protected Routes for Authenticated Users */}
          <Route element={<ProtectedRoute />}>
            <Route element={<AdminLayout />}>
              <Route path="/profile" element={<ProfilePage />} />

              <Route
                element={
                  <ProtectedRoute
                    requiredAnyPermissions={[
                      'user.manage',
                      'role.manage',
                      'setting.manage',
                      'llm.manage',
                      'admin.access',
                    ]}
                  />
                }
              >
                <Route path="/admin" element={<DashboardPage />} />
                <Route path="/admin/dashboard" element={<Navigate to="/admin" replace />} />
                <Route path="/dashboard" element={<Navigate to="/admin" replace />} />
              </Route>

              <Route
                element={
                  <ProtectedRoute
                    requiredAnyPermissions={['user.manage', 'admin.access']}
                  />
                }
              >
                <Route path="/admin/users" element={<UsersListPage />} />
                <Route path="/users" element={<Navigate to="/admin/users" replace />} />
              </Route>

              <Route
                element={
                  <ProtectedRoute
                    requiredAnyPermissions={['role.manage', 'admin.access']}
                  />
                }
              >
                <Route path="/admin/roles" element={<RolesListPage />} />
                <Route path="/roles" element={<Navigate to="/admin/roles" replace />} />
              </Route>

              <Route
                element={
                  <ProtectedRoute
                    requiredAnyPermissions={[
                      'llm.manage',
                      'setting.manage',
                      'admin.access',
                    ]}
                  />
                }
              >
                <Route path="/admin/llm" element={<LLMManagePage />} />
                <Route path="/admin/agent" element={<AgentManagePage />} />
              </Route>

              <Route
                element={
                  <ProtectedRoute
                    requiredAnyPermissions={['chat.manage', 'admin.access']}
                  />
                }
              >
                <Route path="/admin/chats" element={<ChatManagePage />} />
              </Route>

              <Route
                element={
                  <ProtectedRoute
                    requiredAnyPermissions={['rag.manage', 'admin.access']}
                  />
                }
              >
                <Route path="/admin/documents" element={<DocumentManagePage />} />
                <Route path="/admin/knowledge-graph" element={<KnowledgeGraphPage />} />
              </Route>

              <Route
                element={
                  <ProtectedRoute
                    requiredAnyPermissions={['job.manage', 'job.read', 'admin.access']}
                  />
                }
              >
                <Route path="/admin/jobs" element={<JobManagePage />} />
              </Route>

              <Route
                element={
                  <ProtectedRoute
                    requiredAnyPermissions={['inquiry.answer', 'inquiry.read', 'inquiry.manage', 'admin.access']}
                  />
                }
              >
                <Route path="/admin/inquiries" element={<InquiryManagePage />} />
              </Route>
            </Route>
          </Route>

          <Route path="/404" element={<NotFoundPage />} />
          <Route path="*" element={<Navigate to="/404" replace />} />
        </Routes>
        </Suspense>
      </BrowserRouter>
    </QueryClientProvider>
  );
}

export default App;
