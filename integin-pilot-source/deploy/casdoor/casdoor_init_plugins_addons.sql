-- Casdoor Sovereign Identity Bootstrap Script
-- Provisioning Organizations, Applications, Providers (Storage, Email, SMS, Captcha, OAuth), Webhooks, Roles, and Permissions

BEGIN;

-- 1. Sovereign Pilot Organization
INSERT INTO public.organization (
    owner, name, created_time, display_name, website_url, logo, logo_dark, favicon,
    has_privilege_consent, password_type, password_salt, password_options,
    password_expire_days, token_retention_days, record_retention_days, country_codes,
    default_avatar, use_permanent_avatar, default_application, default_token_format,
    user_types, tags, languages, init_score, enable_soft_deletion, is_profile_public,
    use_email_as_username, enable_tour, disable_signin, disable_console,
    account_menu, account_items, dcr_policy, org_balance, user_balance, balance_credit
) VALUES (
    'admin', 'integin-pilot', '2026-10-11T00:00:00Z', 'INTEGIN Sovereign Pilot Organization',
    'http://127.0.0.1:80', 'https://cdn.casbin.org/img/casbin.svg', '', 'https://cdn.casbin.org/img/casbin/favicon.ico',
    false, 'bcrypt', '79a6c77e06332d5a809a', '["AtLeast8"]',
    0, 0, 0, '["US","ES","FR","DE","GB","CN","JP","KR","VN","ID","SG","IN"]',
    'https://cdn.casbin.org/img/casbin.svg', false, 'integin-live-matrix', 'JWT',
    '[]', '[]', '["en","es","fr","de","ja","zh"]', 2000, false, false,
    false, false, false, false,
    'disabled', (SELECT account_items FROM public.organization WHERE name = 'built-in' LIMIT 1),
    'disabled', 0, 0, 0
)
ON CONFLICT (owner, name) DO UPDATE SET
    display_name = EXCLUDED.display_name,
    default_application = EXCLUDED.default_application,
    default_token_format = EXCLUDED.default_token_format;

-- 2. Sovereign Providers (Plugins & Addons)

-- 2a. Storage Provider Plugin: RustFS Sovereign S3
INSERT INTO public.provider (
    owner, name, created_time, display_name, category, type, sub_type, method,
    client_id, client_secret, endpoint, intranet_endpoint, bucket, region_id,
    disable_ssl, enable_proxy, enable_pkce
) VALUES (
    'admin', 'provider_storage_rustfs', '2026-10-11T00:00:00Z', 'RustFS Sovereign S3 Storage',
    'Storage', 'Amazon S3', '', '',
    'sovereign_rustfs_admin', 'sovereign_rustfs_secret_key_643',
    'http://appliance-rustfs:9000', 'http://appliance-rustfs:9000',
    'evidence', 'sovereign-edge',
    true, false, false
)
ON CONFLICT (owner, name) DO UPDATE SET
    display_name = EXCLUDED.display_name,
    client_id = EXCLUDED.client_id,
    client_secret = EXCLUDED.client_secret,
    endpoint = EXCLUDED.endpoint,
    bucket = EXCLUDED.bucket;

-- 2b. Email Provider Plugin: Sovereign SMTP
INSERT INTO public.provider (
    owner, name, created_time, display_name, category, type, host, port,
    disable_ssl, title, content, enable_proxy, enable_pkce
) VALUES (
    'admin', 'provider_email_smtp', '2026-10-11T00:00:00Z', 'Sovereign Appliance SMTP',
    'Email', 'SMTP', '127.0.0.1', 25,
    true, 'INTEGIN Security Verification', 'Your verification code is: %s',
    false, false
)
ON CONFLICT (owner, name) DO UPDATE SET
    display_name = EXCLUDED.display_name,
    host = EXCLUDED.host,
    port = EXCLUDED.port;

-- 2c. SMS Provider Plugin: Sovereign OTP / SMS
INSERT INTO public.provider (
    owner, name, created_time, display_name, category, type,
    title, content, enable_proxy, enable_pkce
) VALUES (
    'admin', 'provider_sms_default', '2026-10-11T00:00:00Z', 'Sovereign SMS Gateway',
    'SMS', 'Mock SMS',
    'INTEGIN SMS', 'Your verification code is: %s',
    false, false
)
ON CONFLICT (owner, name) DO UPDATE SET
    display_name = EXCLUDED.display_name;

-- 2d. OAuth Provider Plugin: Upstream Enterprise OIDC Broker
INSERT INTO public.provider (
    owner, name, created_time, display_name, category, type,
    client_id, client_secret, custom_auth_url, custom_token_url, custom_user_info_url,
    scopes, enable_proxy, enable_pkce
) VALUES (
    'admin', 'provider_oidc_upstream', '2026-10-11T00:00:00Z', 'Upstream Enterprise OIDC Broker',
    'OAuth', 'OpenID',
    'upstream-oidc-client', 'upstream-oidc-secret',
    'http://127.0.0.1:8080/oauth/authorize', 'http://127.0.0.1:8080/oauth/token', 'http://127.0.0.1:8080/oauth/userinfo',
    'openid profile email', false, true
)
ON CONFLICT (owner, name) DO UPDATE SET
    display_name = EXCLUDED.display_name,
    client_id = EXCLUDED.client_id,
    client_secret = EXCLUDED.client_secret;

-- 3. Webhook Addon: Audit Events Stream
INSERT INTO public.webhook (
    owner, name, created_time, organization, url, method, content_type,
    events, is_enabled, max_retries, retry_interval, use_exponential_backoff
) VALUES (
    'admin', 'webhook_integin_audit', '2026-10-11T00:00:00Z', 'integin-pilot',
    'http://appliance-integin-server:8080/api/v1/auth/events', 'POST', 'application/json',
    '["signup","login","logout","update-user","delete-user"]', true, 3, 30, true
)
ON CONFLICT (owner, name) DO UPDATE SET
    url = EXCLUDED.url,
    is_enabled = EXCLUDED.is_enabled,
    events = EXCLUDED.events;

-- 4. Sovereign Applications

-- 4a. integin-live-matrix Application
INSERT INTO public.application (
    owner, name, created_time, display_name, organization, cert,
    client_id, client_secret, redirect_uris, grant_types,
    token_format, token_signing_method, enable_password, enable_sign_up, enable_web_authn,
    expire_in_hours, refresh_expire_in_hours, cookie_expire_in_hours,
    signin_methods, providers
) VALUES (
    'admin', 'integin-live-matrix', '2026-10-11T00:00:00Z', 'INTEGIN Live Matrix Sovereign Application',
    'integin-pilot', 'cert-built-in',
    'integin-live-matrix', 'integin-live-matrix-secret-pilot',
    '["http://127.0.0.1:18080/callback","http://127.0.0.1:80/callback","http://127.0.0.1:80/auth/callback","http://localhost:8080/callback"]',
    '["authorization_code","client_credentials","password","refresh_token"]',
    'JWT', 'RS256', true, true, true,
    168, 720, 24,
    '[{"name":"Password","displayName":"Password","rule":"All"}]',
    '[{"name":"provider_captcha_default","canSignUp":true,"alertWeight":0,"rule":"None"},{"name":"provider_storage_rustfs","canSignUp":false,"alertWeight":0,"rule":"None"}]'
)
ON CONFLICT (owner, name) DO UPDATE SET
    organization = EXCLUDED.organization,
    cert = EXCLUDED.cert,
    client_id = EXCLUDED.client_id,
    client_secret = EXCLUDED.client_secret,
    redirect_uris = EXCLUDED.redirect_uris,
    grant_types = EXCLUDED.grant_types,
    token_format = EXCLUDED.token_format,
    token_signing_method = EXCLUDED.token_signing_method,
    enable_web_authn = EXCLUDED.enable_web_authn,
    providers = EXCLUDED.providers;

-- 4b. Update built-in application with full grant types and RS256 token signing
UPDATE public.application
SET
    grant_types = '["client_credentials","password","authorization_code","refresh_token"]',
    token_format = 'JWT',
    token_signing_method = 'RS256',
    enable_web_authn = true,
    cert = 'cert-built-in',
    redirect_uris = '["http://127.0.0.1:18181/callback","http://127.0.0.1:80/casdoor/callback"]'
WHERE name = 'app-built-in';

-- 5. RBAC Roles Addon
INSERT INTO public.role (
    owner, name, created_time, display_name, description, users, groups, roles, domains, is_enabled
) VALUES
(
    'admin', 'role_integin_admin', '2026-10-11T00:00:00Z',
    'INTEGIN System Administrator', 'Full control over INTEGIN appliance identity and tenants',
    '["integin-pilot/admin"]', '[]', '[]', '[]', true
),
(
    'admin', 'role_integin_operator', '2026-10-11T00:00:00Z',
    'INTEGIN Operational Engineer', 'Operational execution across work orders and device trust',
    '["integin-pilot/operator"]', '[]', '[]', '[]', true
),
(
    'admin', 'role_integin_auditor', '2026-10-11T00:00:00Z',
    'INTEGIN Cryptographic Auditor', 'Audit inspection across event logs and evidence chain',
    '["integin-pilot/auditor"]', '[]', '[]', '[]', true
)
ON CONFLICT (owner, name) DO UPDATE SET
    display_name = EXCLUDED.display_name,
    users = EXCLUDED.users,
    is_enabled = EXCLUDED.is_enabled;

-- 6. RBAC Permissions Addon
INSERT INTO public.permission (
    owner, name, created_time, display_name, description,
    users, groups, roles, domains, model, adapter, resource_type,
    resources, actions, effect, is_enabled, submitter, approver, approve_time, state
) VALUES
(
    'admin', 'permission_integin_admin', '2026-10-11T00:00:00Z',
    'Admin Full Authority', 'Unrestricted administrative access to all system APIs',
    '[]', '[]', '["admin/role_integin_admin"]', '[]',
    'built-in/user-model-built-in', '', 'Application',
    '["*"]', '["*"]', 'Allow', true,
    'admin', 'admin', '2026-10-11T00:00:00Z', 'Approved'
),
(
    'admin', 'permission_integin_operator', '2026-10-11T00:00:00Z',
    'Operator Operational Authority', 'Access to work orders and certificates',
    '[]', '[]', '["admin/role_integin_operator"]', '[]',
    'built-in/user-model-built-in', '', 'Application',
    '["work_orders/*","certificates/*"]', '["Read","Write","Execute"]', 'Allow', true,
    'admin', 'admin', '2026-10-11T00:00:00Z', 'Approved'
),
(
    'admin', 'permission_integin_auditor', '2026-10-11T00:00:00Z',
    'Auditor Verification Authority', 'Read-only access to audit logs and evidence',
    '[]', '[]', '["admin/role_integin_auditor"]', '[]',
    'built-in/user-model-built-in', '', 'Application',
    '["events/*","evidence/*","audit/*"]', '["Read"]', 'Allow', true,
    'admin', 'admin', '2026-10-11T00:00:00Z', 'Approved'
)
ON CONFLICT (owner, name) DO UPDATE SET
    roles = EXCLUDED.roles,
    resources = EXCLUDED.resources,
    actions = EXCLUDED.actions,
    is_enabled = EXCLUDED.is_enabled;

-- 7. Provision Standard Users for Organization integin-pilot
INSERT INTO public.user (
    owner, name, created_time, updated_time, display_name,
    password, password_salt, password_type, email, email_verified,
    is_admin, is_forbidden, is_deleted, signup_application,
    roles, permissions, groups
) VALUES
(
    'integin-pilot', 'admin', '2026-10-11T00:00:00Z', '2026-10-11T00:00:00Z', 'INTEGIN Pilot Admin',
    '$2a$10$qJs1SlTzjKu/9Qtnnv3gfuQeiI8Gei04R5BoBWyu6..6x4.4UvjfS', '79a6c77e06332d5a809a', 'bcrypt',
    'admin@integin.local', true, true, false, false, 'integin-live-matrix',
    '["admin/role_integin_admin"]', '[]', '[]'
),
(
    'integin-pilot', 'operator', '2026-10-11T00:00:00Z', '2026-10-11T00:00:00Z', 'INTEGIN Field Operator',
    '$2a$10$qJs1SlTzjKu/9Qtnnv3gfuQeiI8Gei04R5BoBWyu6..6x4.4UvjfS', '79a6c77e06332d5a809a', 'bcrypt',
    'operator@integin.local', true, false, false, false, 'integin-live-matrix',
    '["admin/role_integin_operator"]', '[]', '[]'
),
(
    'integin-pilot', 'auditor', '2026-10-11T00:00:00Z', '2026-10-11T00:00:00Z', 'INTEGIN Compliance Auditor',
    '$2a$10$qJs1SlTzjKu/9Qtnnv3gfuQeiI8Gei04R5BoBWyu6..6x4.4UvjfS', '79a6c77e06332d5a809a', 'bcrypt',
    'auditor@integin.local', true, false, false, false, 'integin-live-matrix',
    '["admin/role_integin_auditor"]', '[]', '[]'
)
ON CONFLICT (owner, name) DO UPDATE SET
    display_name = EXCLUDED.display_name,
    email = EXCLUDED.email,
    roles = EXCLUDED.roles;

COMMIT;
