-- Seed System Roles
INSERT INTO roles (id, name, description, is_system, is_admin, is_banned, auto_assign, position)
VALUES 
    ('01920000-0000-7000-8000-000000000001', 'ADMIN', 'Quản trị viên toàn hệ thống', 1, 1, 0, 0, 100),
    ('01920000-0000-7000-8000-000000000002', 'ADVISOR', 'Cán bộ Cố vấn học vụ & Phòng đào tạo', 1, 0, 0, 0, 80),
    ('01920000-0000-7000-8000-000000000003', 'STUDENT', 'Sinh viên Đại học Thăng Long', 1, 0, 0, 1, 50),
    ('01920000-0000-7000-8000-000000000004', 'GUEST', 'Khách vãng lai chưa đăng nhập', 1, 0, 0, 0, 10),
    ('01920000-0000-7000-8000-000000000005', 'BANNED', 'Tài khoản bị khóa', 1, 0, 1, 0, -1)
ON CONFLICT(name) DO UPDATE SET
    description = excluded.description,
    is_system = excluded.is_system,
    is_admin = excluded.is_admin,
    is_banned = excluded.is_banned,
    position = excluded.position;

-- Seed Permissions
INSERT INTO permissions (key, description) VALUES
    ('regulation.read', 'Tra cứu và đọc quy chế đào tạo'),
    ('regulation.write', 'Thêm, sửa, xóa quy chế học vụ'),
    ('curriculum.read', 'Xem khung chương trình và cây môn học'),
    ('curriculum.manage', 'Quản lý danh mục môn học, tiên quyết, thay thế'),
    ('crawler.trigger', 'Kích hoạt job cào dữ liệu từ website trường'),
    ('audit.read', 'Xem log bảo mật và lịch sử can thiệp hệ thống'),
    ('chat.ask', 'Gửi tin nhắn hỏi đáp cố vấn học vụ'),
    ('chat.persist', 'Lưu trữ và khôi phục lịch sử chat'),
    ('admin.access', 'Truy cập trang quản trị admin'),
    ('user.manage', 'Quản lý tài khoản sinh viên và người dùng'),
    ('role.manage', 'Quản lý danh sách vai trò và phân quyền'),
    ('setting.manage', 'Cấu hình hệ thống, AI model và tham số')
ON CONFLICT(key) DO UPDATE SET
    description = excluded.description;

-- Permissions for GUEST
INSERT INTO role_permissions (id, role_id, permission_key, effect, conditions_json)
SELECT 'rp-guest-' || p.key, r.id, p.key, 'allow', '{}'
FROM roles r
JOIN permissions p ON p.key IN ('regulation.read', 'chat.ask')
WHERE r.name = 'GUEST'
ON CONFLICT(role_id, permission_key) DO NOTHING;

-- Permissions for STUDENT
INSERT INTO role_permissions (id, role_id, permission_key, effect, conditions_json)
SELECT 'rp-student-' || p.key, r.id, p.key, 'allow', '{}'
FROM roles r
JOIN permissions p ON p.key IN (
    'regulation.read',
    'curriculum.read',
    'chat.ask',
    'chat.persist'
)
WHERE r.name = 'STUDENT'
ON CONFLICT(role_id, permission_key) DO NOTHING;

-- Permissions for ADVISOR
INSERT INTO role_permissions (id, role_id, permission_key, effect, conditions_json)
SELECT 'rp-advisor-' || p.key, r.id, p.key, 'allow', '{}'
FROM roles r
JOIN permissions p ON p.key IN (
    'regulation.read',
    'regulation.write',
    'curriculum.read',
    'curriculum.manage',
    'chat.ask',
    'chat.persist',
    'audit.read'
)
WHERE r.name = 'ADVISOR'
ON CONFLICT(role_id, permission_key) DO NOTHING;

-- Permissions for ADMIN (all permissions)
INSERT INTO role_permissions (id, role_id, permission_key, effect, conditions_json)
SELECT 'rp-admin-' || p.key, r.id, p.key, 'allow', '{}'
FROM roles r
CROSS JOIN permissions p
WHERE r.name = 'ADMIN'
ON CONFLICT(role_id, permission_key) DO UPDATE SET
    effect = 'allow',
    conditions_json = '{}';
