// eslint-disable-next-line no-unused-vars
function closeDialog(event) {
    if (event) {
        event.preventDefault();
        event.stopPropagation();
    }

    // Close closest dialog
    const dialog = event.target.closest('dialog');
    if (dialog) {
        dialog.close();
    }
}
