// Wait for DOM to be ready before initializing
document.addEventListener('DOMContentLoaded', function() {
    // Configure toastr with global settings
    if (typeof toastr === 'undefined') {
        console.error('Toastr library not loaded');
        return;
    }
    toastr.options = {
        "closeButton": true,
        "debug": false,
        "newestOnTop": true,
        "progressBar": true,
        "positionClass": "toast-top-right",
        "preventDuplicates": true,
        "onclick": null,
        "showDuration": "300",
        "hideDuration": "1000",
        "timeOut": "3000",
        "extendedTimeOut": "1000",
        "showEasing": "swing",
        "hideEasing": "linear",
        "showMethod": "fadeIn",
        "hideMethod": "fadeOut"
    };

    // Global HTMX event listeners for API responses
    document.body.addEventListener('htmx:afterRequest', function(event) {
    const xhr = event.detail.xhr;
    const target = event.detail.target;

    // Only show success toasts for API requests (not component loads)
    if (!event.detail.pathInfo.requestPath.startsWith('/api/')) {
        return;
    }

    // Check if the request was successful
    if (xhr.status >= 200 && xhr.status < 300) {
        const method = event.detail.requestConfig.verb.toUpperCase();

        // Show appropriate success messages based on the request type
        if (event.detail.pathInfo.requestPath.includes('/files')) {
            if (method === 'POST') {
                toastr.success('File(s) uploaded successfully');
            } else if (method === 'PUT') {
                toastr.success('File moved/renamed successfully');
            } else if (method === 'DELETE') {
                toastr.success('File(s) deleted successfully');
            }
        } else if (event.detail.pathInfo.requestPath.includes('/folder')) {
            if (method === 'POST') {
                toastr.success('Folder created successfully');
            }
        } else if (event.detail.pathInfo.requestPath.includes('/calendar/events')) {
            if (method === 'POST') {
                toastr.success('Event created successfully');
            } else if (method === 'PUT') {
                toastr.success('Event updated successfully');
            } else if (method === 'DELETE') {
                toastr.success('Event deleted successfully');
            }
        } else if (event.detail.pathInfo.requestPath.includes('/docs')) {
            if (method === 'POST') {
                toastr.success('Document saved successfully');
            }
        }
    }
});

// Global error handler for failed HTMX requests
document.body.addEventListener('htmx:responseError', function(event) {
    const xhr = event.detail.xhr;
    const target = event.detail.target;

    // Only show error toasts for API requests
    if (!event.detail.pathInfo.requestPath.startsWith('/api/')) {
        return;
    }

    let errorMessage = 'Request failed';

    // Try to extract error message from response
    try {
        const response = JSON.parse(xhr.responseText);
        if (response.error) {
            errorMessage = response.error;
        } else if (response.message) {
            errorMessage = response.message;
        }
    } catch (e) {
        // Use status text if JSON parsing fails
        if (xhr.statusText) {
            errorMessage = `${errorMessage}: ${xhr.statusText}`;
        }
    }

    toastr.error(errorMessage);
});

    // Network error handler for HTMX requests
    document.body.addEventListener('htmx:sendError', function(event) {
        toastr.error('Network error: Unable to reach server');
    });

    // Timeout handler for HTMX requests
    document.body.addEventListener('htmx:timeout', function(event) {
        toastr.error('Request timed out');
    });
});
