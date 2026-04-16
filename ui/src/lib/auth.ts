import { UserManager, WebStorageStateStore } from 'oidc-client-ts';

interface AuthConfig {
  provider: string;
  issuer: string;
  clientId: string;
  clientSecret: string;
  scopes: string;
}

let _userManager: UserManager;
let _config: AuthConfig;

export async function initAuth(): Promise<UserManager | null> {
  const resp = await fetch('/api/auth/config');
  if (!resp.ok) {
    throw new Error(`Failed to fetch auth config: ${resp.status} ${resp.statusText}`);
  }
  _config = await resp.json();

  if (_config.provider === 'none') {
    return null;
  }

  if (_config.provider !== 'oidc') {
    throw new Error(`Unsupported auth provider: ${_config.provider}`);
  }

  const settings: ConstructorParameters<typeof UserManager>[0] = {
    authority: _config.issuer,
    client_id: _config.clientId,
    redirect_uri: `${window.location.origin}/callback`,
    post_logout_redirect_uri: window.location.origin,
    scope: _config.scopes,
    response_type: 'code',
    loadUserInfo: true,
    userStore: new WebStorageStateStore({ store: window.localStorage })
  };
  if (_config.clientSecret) {
    settings.client_secret = _config.clientSecret;
  }
  _userManager = new UserManager(settings);

  return _userManager;
}

export function getUserManager(): UserManager {
  if (!_userManager) {
    throw new Error('Auth not initialized — call initAuth() first');
  }
  return _userManager;
}
