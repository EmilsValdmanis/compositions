import { requestBackend } from "./backend-request";

type AuthSession = {
  user: {
    id: string;
    name: string;
    email: string;
    image: string;
    isAdmin: boolean;
    onboardingVersion: number;
    requiredOnboardingVersion: number;
  };
};

export async function readRequestSession(): Promise<AuthSession | null> {
  const response = await requestBackend("/auth/session", { forwardSession: true });
  if (!response.ok) {
    throw new Error(`failed to load session: ${response.status}`);
  }
  return (await response.json()) as AuthSession | null;
}
