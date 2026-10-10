/**
 * Casdoor OIDC PKCE Authentication Module for INTEGIN Edge & Field UI
 * Sovereign authentication via local HAProxy ingress gateway.
 */

export interface CasdoorAuthConfig {
  endpoint: string;
  clientId: string;
  organizationName: string;
  appName: string;
  redirectUri: string;
}

export const defaultCasdoorConfig: CasdoorAuthConfig = {
  endpoint: window.location.origin, // Routed seamlessly through HAProxy
  clientId: "integin-live-matrix",
  organizationName: "integin-pilot",
  appName: "integin-live-matrix",
  redirectUri: `${window.location.origin}/auth/callback`,
};

function generateRandomString(length: number): string {
  const chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-._~";
  const array = new Uint8Array(length);
  window.crypto.getRandomValues(array);
  return Array.from(array, (dec) => chars[dec % chars.length]).join("");
}

async function sha256(plain: string): Promise<ArrayBuffer> {
  const encoder = new TextEncoder();
  const data = encoder.encode(plain);
  return window.crypto.subtle.digest("SHA-256", data);
}

function base64UrlEncode(buffer: ArrayBuffer): string {
  const bytes = new Uint8Array(buffer);
  let binary = "";
  for (let i = 0; i < bytes.byteLength; i++) {
    binary += String.fromCharCode(bytes[i]);
  }
  return btoa(binary).replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/, "");
}

/**
 * Initiates Casdoor OIDC PKCE Flow
 */
export async function loginWithCasdoor(config: CasdoorAuthConfig = defaultCasdoorConfig): Promise<void> {
  const verifier = generateRandomString(64);
  const challenge = base64UrlEncode(await sha256(verifier));
  const state = generateRandomString(32);

  sessionStorage.setItem("casdoor_code_verifier", verifier);
  sessionStorage.setItem("casdoor_auth_state", state);

  const authUrl = new URL(`${config.endpoint}/login/oauth/authorize`);
  authUrl.searchParams.set("client_id", config.clientId);
  authUrl.searchParams.set("response_type", "code");
  authUrl.searchParams.set("redirect_uri", config.redirectUri);
  authUrl.searchParams.set("scope", "openid profile email");
  authUrl.searchParams.set("state", state);
  authUrl.searchParams.set("code_challenge", challenge);
  authUrl.searchParams.set("code_challenge_method", "S256");

  window.location.href = authUrl.toString();
}

/**
 * Handles redirect callback from Casdoor and exchanges authorization code for JWT
 */
export async function handleCasdoorCallback(config: CasdoorAuthConfig = defaultCasdoorConfig): Promise<{ accessToken: string; idToken: string } | null> {
  const urlParams = new URLSearchParams(window.location.search);
  const code = urlParams.get("code");
  const state = urlParams.get("state");
  const storedState = sessionStorage.getItem("casdoor_auth_state");
  const verifier = sessionStorage.getItem("casdoor_code_verifier");

  if (!code || !verifier || state !== storedState) {
    return null;
  }

  const tokenUrl = `${config.endpoint}/api/login/oauth/access_token`;
  const body = new URLSearchParams({
    grant_type: "authorization_code",
    client_id: config.clientId,
    code,
    redirect_uri: config.redirectUri,
    code_verifier: verifier,
  });

  const response = await fetch(tokenUrl, {
    method: "POST",
    headers: { "Content-Type": "application/x-www-form-urlencoded" },
    body: body.toString(),
  });

  if (!response.ok) {
    throw new Error(`Casdoor token exchange failed: ${response.statusText}`);
  }

  const data = await response.json();
  sessionStorage.removeItem("casdoor_code_verifier");
  sessionStorage.removeItem("casdoor_auth_state");
  localStorage.setItem("integin_access_token", data.access_token);
  localStorage.setItem("integin_id_token", data.id_token);

  return {
    accessToken: data.access_token,
    idToken: data.id_token,
  };
}
