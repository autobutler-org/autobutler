// eslint-disable-next-line no-unused-vars
function closeDialog(event) {
    // Close closest dialog
    const dialog = event.target.closest('dialog');
    if (dialog) {
        dialog.close();
    }
}
