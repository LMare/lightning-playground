

/**
 * Global replacement of the native confirm() used by hx-confirm:
 * shows the #confirm-dialog modal and issues the request only on confirmation.
 */
document.addEventListener("htmx:confirm", function (evt) {
	const question = evt.detail.question;
	if (!question) {
		return;
	}
	const dialog = document.getElementById("confirm-dialog");
	if (!dialog || typeof dialog.showModal !== "function") {
		return;
	}
	evt.preventDefault();

	document.getElementById("confirm-dialog-message").textContent = question;
	const labelHolder = evt.detail.elt && evt.detail.elt.closest("[data-confirm-label]");
	const label = labelHolder ? labelHolder.getAttribute("data-confirm-label") : "";
	document.getElementById("confirm-dialog-ok").textContent = label || "Confirm";

	const previousFocus = document.activeElement;
	dialog.returnValue = "";
	dialog.addEventListener("close", function () {
		if (dialog.returnValue === "confirm") {
			evt.detail.issueRequest(true);
		}
		if (previousFocus && previousFocus.isConnected && typeof previousFocus.focus === "function") {
			previousFocus.focus();
		}
	}, { once: true });
	dialog.showModal();
});

/**
 * Closes the unlock dialog of the node page once a wallet is unlocked (HX-Trigger sent by the server).
 * Registered once here, not on every navigation.
 */
document.addEventListener("wallet-unlocked", function () {
	const dialog = document.getElementById("unlock-dialog");
	if (dialog && dialog.open) {
		dialog.close();
	}
});
