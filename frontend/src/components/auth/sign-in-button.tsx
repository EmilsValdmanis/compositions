import { useTransition } from "react";
import { Login01Icon } from "@hugeicons/core-free-icons";
import { HugeiconsIcon } from "@hugeicons/react";
import { Spinner } from "#/components/ui/spinner";
import { Button } from "#/components/ui/button";
import { authClient } from "#/lib/auth-client";

export default function SignInButton() {
  const [isPending, startTransition] = useTransition();

  const handleGoogleSignIn = () => {
    startTransition(() => {
      void authClient.signIn.social({ provider: "google" });
    });
  };

  return (
    <Button onClick={handleGoogleSignIn} disabled={isPending} size="lg">
      {isPending ? (
        <>
          <Spinner data-icon="inline-start" /> Authenticating
        </>
      ) : (
        <>
          Sign in <HugeiconsIcon icon={Login01Icon} />
        </>
      )}
    </Button>
  );
}
