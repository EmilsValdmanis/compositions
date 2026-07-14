import { useTransition } from "react";
import { Login01Icon } from "@hugeicons/core-free-icons";
import { HugeiconsIcon } from "@hugeicons/react";
import { toast } from "sonner";
import { Spinner } from "#/components/ui/spinner";
import { Button } from "#/components/ui/button";
import { authClient } from "#/lib/auth-client";
import { m } from "#/paraglide/messages.js";

export default function SignInButton({ returnTo }: { returnTo?: string }) {
  const [isPending, startTransition] = useTransition();

  const handleGoogleSignIn = () => {
    startTransition(() => {
      void authClient.signIn.social({ provider: "google", returnTo }).catch(() => {
        toast.error(m.error_sign_in_start());
      });
    });
  };

  return (
    <Button onClick={handleGoogleSignIn} disabled={isPending} size="lg">
      {isPending ? (
        <>
          <Spinner data-icon="inline-start" /> {m.authenticating()}
        </>
      ) : (
        <>
          {m.sign_in()} <HugeiconsIcon icon={Login01Icon} />
        </>
      )}
    </Button>
  );
}
