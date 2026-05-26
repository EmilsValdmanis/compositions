import { useTransition } from "react";
import { Login01Icon } from "@hugeicons/core-free-icons";
import { HugeiconsIcon } from "@hugeicons/react";
import { toast } from "sonner";
import { Spinner } from "#/components/ui/spinner";
import { Button } from "#/components/ui/button";
import { authClient } from "#/lib/auth-client";

export default function SignInButton() {
  const [isPending, startTransition] = useTransition();

  const handleGoogleSignIn = () => {
    startTransition(() => {
      void authClient.signIn.social({ provider: "google" }).catch((error: unknown) => {
        const message = error instanceof Error ? error.message : "failed to start sign in";
        toast.error(message);
      });
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
