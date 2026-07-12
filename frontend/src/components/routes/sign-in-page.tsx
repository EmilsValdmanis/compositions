import SignInButton from "#/components/auth/sign-in-button";
import { Route } from "#/routes/_auth/sign-in";

export function SignInPage() {
  const { returnTo } = Route.useSearch();

  return <SignInButton returnTo={returnTo} />;
}
