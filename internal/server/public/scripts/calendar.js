function showEventDialog(eventId) {
    console.log("Showing dialog for event ID:", eventId);
    const dialog = document.getElementById(`event-dialog-${eventId}`);
    if (dialog) {
        dialog.showModal();
    }
}

function closeEventDialog(eventId) {
    console.log("Closing dialog for event ID:", eventId);
    const dialog = document.getElementById(`event-dialog-${eventId}`);
    if (dialog) {
        dialog.close();
    }
}
