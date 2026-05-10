import { HugeiconsIcon } from "@hugeicons/react";
import { Button } from "../ui/button";
import { Login01Icon } from "@hugeicons/core-free-icons";
import { authClient } from "#/lib/auth-client";
import { Spinner } from "../ui/spinner";
import { useState } from "react";

export default function SignInButton() {
  const [isPending, setIsPending] = useState<boolean>(false);

  const handleGoogleSignIn = async () => {
    setIsPending(true);
    await authClient.signIn.social({ provider: "google" });
  };

  return (
    <Button onClick={handleGoogleSignIn} disabled={isPending}>
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
