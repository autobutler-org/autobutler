function preventDefault(event) {
    if (event) {
        event.preventDefault();
        event.stopPropagation();
        event.stopImmediatePropagation();
    }
    return false;
}

function showEventDialog(event, eventId) {
    preventDefault(event);
    const dialog = document.getElementById(`event-dialog-${eventId}`);
    if (dialog) {
        dialog.showModal();
    }
    return false;
}

function closeModal(event, eventId) {
    preventDefault(event);
    const dialog = event.currentTarget.closest("dialog");
    if (dialog) {
        dialog.close();
    }
    return false;
}

function newCalendarEvent(event) {
    preventDefault(event);
    const target = event.currentTarget;
    if (!target) {
        return false;
    }
    const td = target.closest("td");
    const year = td.getAttribute("data-year");
    const month = td.getAttribute("data-month");
    const day = td.getAttribute("data-day");

    const dialog = document.getElementById("new-event-dialog");
    if (dialog) {
        dialog.showModal();
        // Optionally, set hidden fields or inputs in the dialog for year, month, day
        document.getElementById("new-event-year").value = year;
        document.getElementById("new-event-month").value = month;
        document.getElementById("new-event-day").value = day;
    }
    return false;
}

function checkNewEventFormInputs(event) {
    const form = document.getElementById("new-event-form");
    const input = event.target;
    if (!form || !input) {
        return;
    }
    const submitButton = form.querySelector('input[type="submit"]');
    if (!submitButton) {
        return;
    }

    // Check if all required inputs are filled
    const requiredInputs = Array.from(form.querySelectorAll("input[required], textarea[required]"));
    submitButton.disabled = requiredInputs.some(input => !input.value);
}
