function copyTextWithSelection(text: string) {
  const textArea = document.createElement("textarea");
  const activeElement =
    document.activeElement instanceof HTMLElement ? document.activeElement : null;

  textArea.value = text;
  textArea.readOnly = true;
  textArea.style.position = "fixed";
  textArea.style.inset = "0";
  textArea.style.opacity = "0";
  textArea.style.pointerEvents = "none";

  document.body.append(textArea);
  textArea.focus();
  textArea.select();
  textArea.setSelectionRange(0, text.length);

  try {
    if (!document.execCommand("copy")) {
      throw new Error("copy command was rejected");
    }
  } finally {
    textArea.remove();
    activeElement?.focus();
  }
}

export async function copyTextToClipboard(text: string) {
  if (navigator.clipboard?.writeText) {
    try {
      await navigator.clipboard.writeText(text);
      return;
    } catch {
      // Fall through for browsers or permission policies that reject Clipboard API access.
    }
  }

  copyTextWithSelection(text);
}
