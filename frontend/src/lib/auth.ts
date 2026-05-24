import { betterAuth } from "better-auth";
import { bearer } from "better-auth/plugins";

export const AUTH_COOKIE_PREFIX = "compositions";
export const AUTH_SESSION_COOKIE_NAME = "session_token";

export const auth = betterAuth({
  baseURL: process.env.BETTER_AUTH_URL,
  advanced: {
    cookiePrefix: AUTH_COOKIE_PREFIX,
    useSecureCookies: process.env.NODE_ENV === "production",
  },
  plugins: [bearer({ requireSignature: true })],
  socialProviders: {
    google: {
      clientId: process.env.GOOGLE_CLIENT_ID,
      clientSecret: process.env.GOOGLE_CLIENT_SECRET,
    },
  },
});
