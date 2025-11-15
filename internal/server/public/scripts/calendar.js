function preventDefault(event) {
    if (event) {
        event.preventDefault();
        event.stopPropagation();
        event.stopImmediatePropagation();
    }
    return false;
}

// eslint-disable-next-line no-unused-vars
function showEventDialog(event, eventId) {
    preventDefault(event);
    const dialog = document.getElementById(`event-dialog-${eventId}`);
    if (dialog) {
        dialog.showModal();

        // Set the viewing context fields after content loads
        setTimeout(() => {
            const viewYear = dialog.getAttribute('data-view-year');
            const viewMonth = dialog.getAttribute('data-view-month');
            const viewYearInput = document.getElementById('view-year');
            const viewMonthInput = document.getElementById('view-month');
            if (viewYearInput && viewMonthInput) {
                viewYearInput.value = viewYear;
                viewMonthInput.value = viewMonth;
            }
        }, 10);
    }
    return false;
}

// eslint-disable-next-line no-unused-vars
function closeModal(event) {
    preventDefault(event);
    const dialog = event.currentTarget.closest('dialog');
    if (dialog) {
        dialog.close();
    }
    return false;
}

// eslint-disable-next-line no-unused-vars
function newCalendarEvent(event) {
    preventDefault(event);
    const target = event.currentTarget;
    if (!target) {
        return false;
    }
    const td = target.closest('td');
    const year = td.getAttribute('data-year');
    const month = td.getAttribute('data-month');
    const day = td.getAttribute('data-day');

    const dialog = document.getElementById('new-event-dialog');
    if (dialog) {
        dialog.showModal();
        // Set the event date fields
        document.getElementById('new-event-year').value = year;
        document.getElementById('new-event-month').value = month;
        document.getElementById('new-event-day').value = day;

        // Set the viewing context fields
        const viewYear = dialog.getAttribute('data-view-year');
        const viewMonth = dialog.getAttribute('data-view-month');
        document.getElementById('view-year').value = viewYear;
        document.getElementById('view-month').value = viewMonth;
    }
    return false;
}

// eslint-disable-next-line no-unused-vars
function checkNewEventFormInputs(event) {
    const form = document.getElementById('new-event-form');
    const input = event.target;
    if (!form || !input) {
        return;
    }
    const submitButton = form.querySelector('input[type="submit"]');
    if (!submitButton) {
        return;
    }

    // Check if all required inputs are filled
    const requiredInputs = Array.from(form.querySelectorAll('input[required], textarea[required]'));
    submitButton.disabled = requiredInputs.some((input) => !input.value);
}
