import { HugeiconsIcon } from "@hugeicons/react";
import { Button } from "../ui/button";
import { Logout02FreeIcons } from "@hugeicons/core-free-icons";
import { useRouter } from "@tanstack/react-router";
import { authClient } from "#/lib/auth-client";
import { useState } from "react";
import { Spinner } from "../ui/spinner";

export default function SignOutButton() {
  const [isPending, setIsPending] = useState<boolean>(false);
  const router = useRouter();

  const handleSignOut = async () => {
    setIsPending(true);
    await authClient.signOut();
    await router.invalidate();
  };
  return (
    <Button variant="destructive" size="icon" onClick={handleSignOut} disabled={isPending}>
      {isPending ? <Spinner /> : <HugeiconsIcon icon={Logout02FreeIcons} />}
    </Button>
  );
}
