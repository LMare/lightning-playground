

/**
 * Add RGAA attributes on the lines of the tables.
 * columnIds lists only the <td> columns (not the row header column).
 */
function enrichRgaaTableRows(tbody, columnIds = []) {
	const rows = tbody.querySelectorAll("tr");
	if (!tbody.id) {
		console.warn("No id on the tbody", tbody);
	}
	const prefixId = tbody.id ? `${tbody.id}-row` : "row";
	rows.forEach((row, rowIndex) => {
		const th = row.querySelector("th");
		let rowId = "";
		if (th) {
			rowId = `${prefixId}-${rowIndex}`;
			th.id = rowId;
		}

		const tds = row.querySelectorAll("td");
		tds.forEach((td, i) => {
			if (td.hasAttribute("colspan")) {
				return;
			}
			const colId = columnIds[i];
			if (colId) {
				td.setAttribute("headers", rowId ? `${rowId} ${colId}` : colId);
			}
		});
	});
}

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
