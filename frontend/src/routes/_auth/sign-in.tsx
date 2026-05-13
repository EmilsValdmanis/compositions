import { createFileRoute } from "@tanstack/react-router";
import SignInButton from "#/components/auth/sign-in-button";

export const Route = createFileRoute("/_auth/sign-in")({
  component: SignIn,
});

function SignIn() {
  return <SignInButton />;
}
