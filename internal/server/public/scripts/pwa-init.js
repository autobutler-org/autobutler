// Register service worker for PWA functionality
if ('serviceWorker' in navigator) {
    window.addEventListener('load', () => {
        navigator.serviceWorker
            .register('/public/scripts/service-worker.js')
            .then((registration) => {
                console.log('Service Worker registered successfully:', registration.scope);
            })
            .catch((error) => {
                console.error('Service Worker registration failed:', error);
            });
    });
}

// Handle PWA install prompt
window.addEventListener('beforeinstallprompt', (e) => {
    // Prevent the mini-infobar from appearing on mobile
    e.preventDefault();
    // Optionally, show your own install button UI here
    console.log('PWA install prompt available');
});

window.addEventListener('appinstalled', () => {
    console.log('PWA installed successfully');
});
