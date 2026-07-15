import SignInButton from "#/components/auth/sign-in-button";

export function SignInPage({ returnTo }: { returnTo?: string }) {
  return <SignInButton returnTo={returnTo} />;
}
