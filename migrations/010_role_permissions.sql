-- 010_role_permissions.sql — cerminkan peta izin kode (internal/rbac/rbac.go) ke DB.
-- Penegakan tetap di kode; tabel ini menjadi cermin audit + basis UI /roles.
-- Idempoten via INSERT IGNORE.
INSERT IGNORE INTO role_permissions(role_id, permission_id)
SELECT r.id, p.id FROM roles r JOIN permissions p ON (
  (r.name='super_admin')
  OR (r.name='office_admin' AND p.name IN ('client.view','client.create','client.edit','identity.view','deed.draft','deed.edit','protocol.view','repertorium.manage','finance.view','template.manage','users.manage','reports.view','office.manage','legalization.process','government.submit'))
  OR (r.name='notary' AND p.name IN ('client.view','client.create','client.edit','identity.view','identity.verify','deed.draft','deed.edit','deed.approve','protocol.view','repertorium.manage','finance.view','template.manage','reports.view','legalization.process','government.submit'))
  OR (r.name='substitute_notary' AND p.name IN ('client.view','identity.view','deed.draft','deed.edit','deed.approve','protocol.view','reports.view','legalization.process'))
  OR (r.name='staff' AND p.name IN ('client.view','client.create','client.edit','identity.view','deed.draft','deed.edit','reports.view','service.request'))
  OR (r.name='finance' AND p.name IN ('client.view','finance.view','finance.manage','reports.view'))
  OR (r.name='doc_operator' AND p.name IN ('client.view','deed.draft','deed.edit','protocol.view','template.manage'))
  OR (r.name='client' AND p.name IN ('service.request'))
  OR (r.name='auditor' AND p.name IN ('client.view','finance.view','protocol.view','reports.view'))
);
