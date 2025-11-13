var MAX_FILE_NAME_LENGTH = 255;

var contextMenuPosition = {
    x: null,
    y: null,
};
// NOTE: Must be global so that it can be removed when the file viewer is closed.
var navigationListener = null;
var loadedBook = null;

// SELECTION STATE
var selectedFiles = [];
var clickTimer = null;
var DOUBLE_CLICK_DELAY = 200; // ms

// VIEW STATE MANAGEMENT
var VIEW_STORAGE_KEY = 'fileExplorerView';

function getViewPreference() {
    return localStorage.getItem(VIEW_STORAGE_KEY) || 'list';
}

function setViewPreference(view) {
    localStorage.setItem(VIEW_STORAGE_KEY, view);
    // Also set cookie so server can read it on regular page loads
    document.cookie = `fileExplorerView=${view}; path=/; max-age=31536000`; // 1 year
}

function switchView(view) {
    setViewPreference(view);

    // Update button states immediately
    updateViewButtonStates(view);

    // Use HTMX to reload the content without a full page refresh
    const currentPath = window.location.pathname;
    htmx.ajax('GET', currentPath, {
        target: '#file-explorer-view-content',
        swap: 'innerHTML'
    });
}

function updateViewButtonStates(activeView) {
    // Get all view switcher buttons
    const viewSwitcher = document.querySelector('.view-switcher');
    if (!viewSwitcher) return;

    const buttons = viewSwitcher.querySelectorAll('button');
    buttons.forEach((button, index) => {
        const views = ['list', 'grid', 'column'];
        const buttonView = views[index];

        if (buttonView === activeView) {
            // Make this button active
            button.classList.remove('btn--secondary');
            button.classList.add('btn--primary');
        } else {
            // Make this button inactive
            button.classList.remove('btn--primary');
            button.classList.add('btn--secondary');
        }
    });
}

// Initialize - sync localStorage to cookie on page load and update button states
document.addEventListener('DOMContentLoaded', function() {
    const view = getViewPreference();
    setViewPreference(view); // Ensures cookie is set
    updateViewButtonStates(view); // Ensure button states match the active view
});

// Send view preference in all HTMX requests via custom header
document.body.addEventListener('htmx:configRequest', function(event) {
    const view = getViewPreference();
    event.detail.headers['X-File-Explorer-View'] = view;
});

function preventDefault(event) {
    if (event) {
        event.preventDefault();
        event.stopPropagation();
    }
}

function debounce(func, wait) {
    let timeout;
    return (...args) => {
        clearTimeout(timeout);
        timeout = setTimeout(() => func.apply(this, args), wait);
    };
}

function showFileDetails(event, fileName) {
    alert(fileName);
}

function closeContextMenu(event, parentNode) {
    preventDefault(event);
    for (const contextMenu of parentNode.querySelectorAll('.context-menu')) {
        contextMenu.style.left = null;
        contextMenu.style.top = null;
        contextMenu.classList.add('hidden');
    }
}

function closeContextMenuFromItem(event) {
    const contextMenu = event.target.closest('.context-menu');
    if (contextMenu) {
        contextMenu.style.left = null;
        contextMenu.style.top = null;
        contextMenu.classList.add('hidden');
    }
}

function openContextMenu(event, parentNode) {
    preventDefault(event);
    clearSelectedFiles();
    const contextMenu = parentNode.querySelector('.context-menu');
    contextMenu.style.left = null;
    contextMenu.style.top = null;
    contextMenu.classList.remove('hidden');
    return contextMenu;
}

function toggleFloatingContextMenu(event, parentNode) {
    preventDefault(event);
    const contextMenu = parentNode.querySelector('.context-menu');
    if (!contextMenu) {
        console.error('Context menu not found in', parentNode);
        return;
    }

    // Close any other open context menus first
    document.querySelectorAll('.context-menu:not(.hidden)').forEach(menu => {
        if (menu !== contextMenu) {
            menu.classList.add('hidden');
        }
    });

    // Toggle this context menu
    const isHidden = contextMenu.classList.contains('hidden');
    contextMenu.classList.toggle('hidden');

    if (isHidden) {
        // Position at click location
        contextMenu.style.left = `${event.clientX}px`;
        contextMenu.style.top = `${event.clientY}px`;

        // Ensure menu doesn't go off screen
        requestAnimationFrame(() => {
            const rect = contextMenu.getBoundingClientRect();
            const viewportWidth = window.innerWidth;
            const viewportHeight = window.innerHeight;

            // Adjust if menu goes off right edge
            if (rect.right > viewportWidth) {
                contextMenu.style.left = `${viewportWidth - rect.width - 10}px`;
            }

            // Adjust if menu goes off bottom edge
            if (rect.bottom > viewportHeight) {
                contextMenu.style.top = `${viewportHeight - rect.height - 10}px`;
            }
        });
    }
}

function toggleFolderInput(event) {
    event.preventDefault();
    const folderInput = document.getElementById('folder-input');
    if (!folderInput.classList.toggle('hidden')) {
        folderInput.focus();
    }
}

// Close context menus when clicking outside
document.addEventListener('click', (event) => {
    // Check if the click is outside any context menu and not on a context menu trigger
    if (!event.target.closest('.context-menu') && !event.target.closest('.context-menu-trigger')) {
        document.querySelectorAll('.context-menu:not(.hidden)').forEach(menu => {
            menu.classList.add('hidden');
            menu.style.left = null;
            menu.style.top = null;
        });
    }
});

function clearFileViewer() {
    if (loadedBook) {
        loadedBook.destroy();
        loadedBook = null;
    }
    const fileViewerContent = document.getElementById('file-viewer-content');
    fileViewerContent.innerHTML = '';
    if (navigationListener) {
        removeEventListener('keydown', navigationListener);
        navigationListener = null;
    }
}

function closeFileViewer(event) {
    preventDefault(event);
    const fileViewer = document.getElementById('file-viewer');
    fileViewer.close();
    clearFileViewer();
}

// SELECTION MANAGEMENT (Google Drive style)

/**
 * Clear all selected files and remove visual selection
 */
function clearSelectedFiles() {
    document.querySelectorAll('.file-node--selected').forEach(node => {
        node.classList.remove('file-node--selected');
    });
    selectedFiles = [];
    updateDownloadButton();
}

/**
 * Select a single file node
 */
function selectFileNode(node) {
    if (!node) return;

    // Add selection class with temporary logging for debugging
    node.classList.add('file-node--selected');

    // Temporarily log for debugging
    if (window.location.search.includes('debug=1')) {
        console.log('Selected node:', node, 'Classes:', node.className);
    }

    const fileName = node.dataset.name;
    if (fileName && !selectedFiles.includes(fileName)) {
        selectedFiles.push(fileName);
    }
    updateDownloadButton();
}

/**
 * Deselect a single file node
 */
function deselectFileNode(node) {
    if (!node) return;
    node.classList.remove('file-node--selected');
    const fileName = node.dataset.name;
    if (fileName) {
        selectedFiles = selectedFiles.filter(name => name !== fileName);
    }
    updateDownloadButton();
}

/**
 * Update the download button state based on selection
 */
function updateDownloadButton() {
    const downloadBtn = document.getElementById('file-download-button');
    if (!downloadBtn) return;

    if (selectedFiles.length > 0) {
        downloadBtn.disabled = false;
        downloadBtn.classList.remove('btn--disabled');
        downloadBtn.classList.add('btn--secondary');
    } else {
        downloadBtn.disabled = true;
        downloadBtn.classList.remove('btn--secondary');
        downloadBtn.classList.add('btn--disabled');
    }
}

/**
 * Handle single click on a file node
 * Single click = select the file (Google Drive style)
 */
function handleFileNodeClick(event, node) {
    // Ignore if clicking on context menu trigger
    if (event.target.closest('.context-menu-trigger') ||
        event.target.closest('.grid-view-context-trigger') ||
        event.target.closest('.column-view-context-trigger')) {
        return;
    }

    // Clear any pending double-click timer
    if (clickTimer) {
        clearTimeout(clickTimer);
        clickTimer = null;
    }

    // Wait to see if this becomes a double-click
    clickTimer = setTimeout(() => {
        clickTimer = null;

        // Single click behavior - toggle selection
        if (event.ctrlKey || event.metaKey) {
            // Ctrl/Cmd+Click: toggle this item's selection
            if (node.classList.contains('file-node--selected')) {
                deselectFileNode(node);
            } else {
                selectFileNode(node);
            }
        } else if (event.shiftKey) {
            // Shift+Click: range selection (future enhancement)
            // For now, just select this item
            clearSelectedFiles();
            selectFileNode(node);
        } else {
            // Regular click: select only this item
            clearSelectedFiles();
            selectFileNode(node);
        }
    }, DOUBLE_CLICK_DELAY);
}

/**
 * Handle double-click on a file node
 * Double-click = navigate/open the file (Google Drive style)
 */
function handleFileNodeDoubleClick(event, node) {
    // Cancel the single-click timer
    if (clickTimer) {
        clearTimeout(clickTimer);
        clickTimer = null;
    }

    preventDefault(event);

    const fileType = node.dataset.fileType;

    if (fileType === 'folder') {
        // Navigate to folder - use the stored href
        const contentCell = node.querySelector('[data-href]');
        const href = contentCell?.dataset.href;
        if (href) {
            // Use HTMX for smooth navigation without page reload
            htmx.ajax('GET', href, {
                target: '#file-explorer-view-content',
                swap: 'innerHTML'
            }).then(() => {
                // Update the browser URL after successful navigation
                window.history.pushState({}, '', href);
                updateBackButton();
            });
        }
    } else {
        // Open file viewer
        const viewerCell = node.querySelector('[data-viewer-path]');
        const viewerPath = viewerCell?.dataset.viewerPath;
        if (viewerPath) {
            const fileViewer = document.getElementById('file-viewer');
            if (fileViewer) {
                fileViewer.showModal();
                htmx.ajax('GET', viewerPath, {
                    target: '#file-viewer-content',
                    swap: 'innerHTML'
                });
            }
        }
    }
}

function supportsDirectoryUpload() {
    const supportsFileSystemAccessAPI = 'getAsFileSystemHandle' in DataTransferItem.prototype;
    const supportsWebkitGetAsEntry = 'webkitGetAsEntry' in DataTransferItem.prototype;
    // NOTE: I have found that none of my browsers support this, and likely is why Google Drive does not support
    // folder upload without a separate input.
    return supportsFileSystemAccessAPI || supportsWebkitGetAsEntry;
}

function activateDropZone(event) {
    preventDefault(event);
    const fileUploadArea = document.getElementById('file-upload-area');
    fileUploadArea.classList.add('bg-blue-600');
    fileUploadArea.classList.remove('bg-gray-800');
}

function deactivateDropZone(event) {
    preventDefault(event);
    const fileUploadArea = document.getElementById('file-upload-area');
    fileUploadArea.classList.remove('bg-blue-600');
    fileUploadArea.classList.add('bg-gray-800');
}

function activateDropZoneOnNode(event) {
    preventDefault(event);
    event.currentTarget.classList.add('bg-blue-600');
}

function deactivateDropZoneOnNode(event) {
    preventDefault(event);
    event.currentTarget.classList.remove('bg-blue-600');
}

function dropOnNode(event, returnDir) {
    preventDefault(event);
    event.currentTarget.classList.remove('bg-blue-600');
    const li = event.currentTarget.closest('li');
    const dropDir = li.dataset.name;
    console.log(`Drop on node: ${dropDir}`);
    return dropFiles(event, `/${dropDir}`, !!returnDir ? returnDir : "/");
}

function downloadSelectedFiles(event, rootDir) {
    preventDefault(event);

    console.log('Download requested. Root dir:', rootDir, 'Selected files:', selectedFiles);

    if (!rootDir) rootDir = '';

    selectedFiles.forEach(fileName => {
        const link = document.createElement('a');
        let cleanFileName = fileName;
        while (cleanFileName.endsWith('/')) {
            cleanFileName = cleanFileName.slice(0, -1);
        }

        // Construct the proper path - ensure no double slashes
        let filePath;
        if (rootDir && rootDir !== '/') {
            // Remove leading slash from rootDir if present
            const cleanRootDir = rootDir.startsWith('/') ? rootDir.slice(1) : rootDir;
            filePath = `/api/v1/files/${cleanRootDir}/${cleanFileName}`;
        } else {
            filePath = `/api/v1/files/${cleanFileName}`;
        }

        console.log('Downloading:', filePath);

        link.href = filePath;
        link.download = cleanFileName;
        document.body.appendChild(link);
        link.click();
        document.body.removeChild(link);
    });
    clearSelectedFiles();
}

function dropFiles(event, rootDir, returnDir) {
    rootDir = rootDir || "";
    returnDir = returnDir || "";
    preventDefault(event);
    const files = event.dataTransfer.files;
    if (files.length > 0) {
        const formData = new FormData();
        for (const file of files) {
            formData.append('files', file);
        }
        const uploadForm = document.getElementById('file-upload-form');
        // NOTE: https://flaviocopes.com/htmx-send-files-using-htmxajax-call/
        htmx.ajax('POST',
            uploadForm.getAttribute('hx-post') + rootDir, {
            values: {
                files: formData.getAll('files'),
                returnDir: returnDir,
            },
            source: uploadForm,
        });
    }
}

function saveQuill(filePath) {
    const delta = quill.getContents();
    fetch(`/api/v1/docs${filePath}`, {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json'
        },
        body: JSON.stringify(delta)
    }).then(response => {
        if (response.ok) {
            toastr.success('Document saved successfully');
        } else {
            return response.text().then(text => {
                toastr.error('Error saving document: ' + (text || response.statusText));
            });
        }
    }).catch(error => {
        console.error('Error saving file:', error);
        toastr.error('Error saving document: ' + error.message);
    });
}

function saveAceEditor(filePath, content) {
    fetch(`/api/v1/files${filePath}`, {
        method: 'POST',
        headers: {
            'Content-Type': 'text/plain'
        },
        body: content
    }).then(response => {
        if (response.ok) {
            toastr.success('File saved successfully');
            console.log('File saved successfully');
        } else {
            return response.text().then(text => {
                toastr.error('Error saving file: ' + (text || response.statusText));
                console.error('Error saving file:', response.statusText);
            });
        }
    }).catch(error => {
        console.error('Error saving file:', error);
        toastr.error('Error saving file: ' + error.message);
    });
}

function moveFile(event, rootDir, fileName) {
    preventDefault(event);
    while (rootDir && rootDir[0] == '/') {
        rootDir = rootDir.slice(1);
    }
    const filePath = `${rootDir}/${fileName}`;

    // Create overlay and dialog HTML
    const overlay = document.createElement('div');
    overlay.className = 'ab-rename-overlay';
    overlay.innerHTML = `
        <div class="ab-rename-dialog">
            <button class="ab-rename-close" aria-label="Close" id="ab-rename-close-btn">
                <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" fill="none" viewBox="0 0 20 20">
                    <path stroke="currentColor" stroke-width="2" stroke-linecap="round" d="M5 5l10 10M15 5l-10 10"></path>
                </svg>
            </button>
            <div class="ab-rename-header">
                <h3 class="ab-rename-title">Rename/Move File</h3>
                <p class="ab-rename-subtitle">Current: ${filePath}</p>
            </div>
            <form class="ab-rename-form" id="ab-rename-form">
                <div class="ab-rename-confirm ab-rename-hidden" id="ab-rename-confirm">
                    <p class="ab-rename-confirm-text">Are you sure you want to rename/move this file?</p>
                    <div class="ab-rename-confirm-path" id="ab-rename-confirm-from">From: ${filePath}</div>
                    <div class="ab-rename-confirm-path" id="ab-rename-confirm-to"></div>
                </div>
                <div class="ab-rename-input-group" id="ab-rename-input-group">
                    <label class="ab-rename-label" for="ab-rename-path-input">
                        New path (including filename and extension):
                    </label>
                    <input
                        type="text"
                        id="ab-rename-path-input"
                        class="ab-rename-input"
                        value="${filePath}"
                        maxlength="${MAX_FILE_NAME_LENGTH}"
                        required
                    />
                </div>
                <div class="ab-rename-actions">
                    <button type="button" class="btn btn--secondary" id="ab-rename-cancel">
                        Cancel
                    </button>
                    <button type="submit" class="btn btn--primary" id="ab-rename-submit">
                        Continue
                    </button>
                </div>
            </form>
        </div>
    `;

    document.body.appendChild(overlay);

    const input = document.getElementById('ab-rename-path-input');
    const form = document.getElementById('ab-rename-form');
    const confirmBox = document.getElementById('ab-rename-confirm');
    const inputGroup = document.getElementById('ab-rename-input-group');
    const submitBtn = document.getElementById('ab-rename-submit');
    const cancelBtn = document.getElementById('ab-rename-cancel');
    const confirmTo = document.getElementById('ab-rename-confirm-to');
    const closeBtn = document.getElementById('ab-rename-close-btn');

    let isConfirming = false;

    // Focus and select the input
    setTimeout(() => {
        input.focus();
        // Select just the filename without extension for easy editing
        const lastSlashIndex = input.value.lastIndexOf('/');
        const lastDotIndex = input.value.lastIndexOf('.');
        if (lastDotIndex > lastSlashIndex) {
            input.setSelectionRange(lastSlashIndex + 1, lastDotIndex);
        } else {
            input.setSelectionRange(lastSlashIndex + 1, input.value.length);
        }
    }, 10);

    // Handle overlay click to close
    overlay.addEventListener('click', (e) => {
        if (e.target === overlay) {
            overlay.remove();
        }
    });

    // Handle overlay click to close
    overlay.addEventListener('click', (e) => {
        if (e.target === overlay) {
            overlay.remove();
        }
    });

    // Handle close button
    closeBtn.addEventListener('click', () => {
        overlay.remove();
    });

    // Handle cancel button
    cancelBtn.addEventListener('click', () => {
        overlay.remove();
    });

    // Handle Escape key
    const escapeHandler = (e) => {
        if (e.key === 'Escape') {
            overlay.remove();
            document.removeEventListener('keydown', escapeHandler);
        }
    };
    document.addEventListener('keydown', escapeHandler);

    // Handle form submission
    form.addEventListener('submit', (e) => {
        e.preventDefault();

        const newFilePath = input.value.trim();

        // Validation
        if (!newFilePath) {
            input.focus();
            return;
        }

        if (newFilePath === filePath) {
            overlay.remove();
            return;
        }

        const newFileName = newFilePath.split('/').pop();
        if (newFileName.length > MAX_FILE_NAME_LENGTH) {
            alert(`File name must be ${MAX_FILE_NAME_LENGTH} characters or less`);
            input.focus();
            return;
        }

        // Show confirmation step
        if (!isConfirming) {
            isConfirming = true;
            inputGroup.classList.add('ab-rename-hidden');
            confirmBox.classList.remove('ab-rename-hidden');
            confirmTo.textContent = `To: ${newFilePath}`;
            submitBtn.textContent = 'Confirm';
            cancelBtn.textContent = 'Go Back';

            // Update cancel button to go back to editing
            cancelBtn.onclick = () => {
                isConfirming = false;
                inputGroup.classList.remove('ab-rename-hidden');
                confirmBox.classList.add('ab-rename-hidden');
                submitBtn.textContent = 'Continue';
                cancelBtn.textContent = 'Cancel';
                cancelBtn.onclick = () => overlay.remove();
                input.focus();
            };
            return;
        }

        // Actually perform the rename/move
        overlay.remove();
        document.removeEventListener('keydown', escapeHandler);

        htmx.ajax('PUT',
            `/api/v1/files/${filePath}`, {
            values: {
                newFilePath: newFilePath,
            },
            target: '#file-explorer',
            swap: 'outerHTML',
        });
    });
}

function newFile(event, rootDir) {
    preventDefault(event);
    const fileName = prompt("Enter the new file name (including extension):");
    if (fileName) {
        if (fileName.length > MAX_FILE_NAME_LENGTH) {
            alert(`File name must be ${MAX_FILE_NAME_LENGTH} characters or less`);
            return;
        }
        const uploadForm = document.getElementById('file-upload-form');
        const formData = new FormData();
        // NOTE: Creating an empty file
        formData.append('files', new Blob([''], { type: 'text/plain' }), fileName);
        htmx.ajax('POST',
            uploadForm.getAttribute('hx-post') + rootDir, {
            values: {
                files: formData.getAll('files'),
                returnDir: rootDir,
            },
            source: uploadForm,
        });
    }
}

function showFolderDetails(event) {
    preventDefault(event);
    alert("Folder details to be implemented.");
}

function navigateToParentAndPreview(event, parentPath, previewPath) {
    preventDefault(event);
    // Use HTMX to navigate to parent (removes child columns) without full page reload
    htmx.ajax('GET', parentPath, {
        target: '#file-explorer-view-content',
        swap: 'innerHTML'
    }).then(function() {
        // After the file explorer updates, load the preview
        htmx.ajax('GET', previewPath, {
            target: '#column-preview-content',
            swap: 'innerHTML'
        });
    });
    // Update the URL
    history.pushState({}, '', parentPath);
}


// SORTING

var currentSortColumn = null;
var currentSortDirection = 'asc'; // 'asc' or 'desc'
var mixedSorting = false; // false = folders first, true = mixed sorting

function sortFiles(column) {
    if (currentSortColumn === column) {
        currentSortDirection = currentSortDirection === 'asc' ? 'desc' : 'asc';
    } else {
        currentSortDirection = 'asc';
    }
    currentSortColumn = column;

    applySorting();
}

function applySorting() {
    const column = currentSortColumn;
    if (!column) return;

    const tbody = document.getElementById('file-explorer-list');
    const rows = Array.from(tbody.querySelectorAll('tr'));

    // Separate the spacer row (last row with "Drop files here...")
    const spacerRow = rows.find(row => row.querySelector('.spacer'));
    const fileRows = rows.filter(row => row !== spacerRow);

    fileRows.sort((a, b) => {
        let aValue, bValue;

        if (column === 'name') {
            aValue = a.dataset.name || '';
            bValue = b.dataset.name || '';

            // Sort folders first, then files (unless mixed sorting is enabled)
            if (!mixedSorting) {
                const aIsFolder = a.querySelector('td:first-child a[href]') !== null;
                const bIsFolder = b.querySelector('td:first-child a[href]') !== null;

                if (aIsFolder && !bIsFolder) return -1;
                if (!aIsFolder && bIsFolder) return 1;
            }

            // Sort alphabetically
            return currentSortDirection === 'asc'
                ? aValue.localeCompare(bValue, undefined, { numeric: true, sensitivity: 'base' })
                : bValue.localeCompare(aValue, undefined, { numeric: true, sensitivity: 'base' });
        } else if (column === 'size') {
            // Sort folders first, then files (unless mixed sorting is enabled)
            if (!mixedSorting) {
                const aIsFolder = a.querySelector('td:first-child a[href]') !== null;
                const bIsFolder = b.querySelector('td:first-child a[href]') !== null;

                if (aIsFolder && !bIsFolder) return -1;
                if (!aIsFolder && bIsFolder) return 1;
            }

            // Extract size text and convert to bytes for comparison
            const aSizeText = a.querySelector('td:nth-child(2)')?.textContent?.trim() || '0 B';
            const bSizeText = b.querySelector('td:nth-child(2)')?.textContent?.trim() || '0 B';

            aValue = parseSize(aSizeText);
            bValue = parseSize(bSizeText);

            return currentSortDirection === 'asc' ? aValue - bValue : bValue - aValue;
        }

        return 0;
    });

    // Clear tbody and re-append sorted rows
    tbody.innerHTML = '';
    fileRows.forEach(row => tbody.appendChild(row));

    // Add spacer row back at the end
    if (spacerRow) {
        tbody.appendChild(spacerRow);
    }
}

function parseSize(sizeText) {
    const units = { 'B': 1, 'KB': 1024, 'MB': 1024 * 1024, 'GB': 1024 * 1024 * 1024, 'TB': 1024 * 1024 * 1024 * 1024 };
    const match = sizeText.match(/^([\d.]+)\s*([A-Z]+)$/);
    if (!match) return 0;

    const value = parseFloat(match[1]);
    const unit = match[2];
    return value * (units[unit] || 1);
}

function updateSortArrows(column) {
    // Hide all arrows first
    const allArrows = document.querySelectorAll('[id$="-sort-asc"], [id$="-sort-desc"]');
    allArrows.forEach(arrow => {
        arrow.classList.add('hidden');
        arrow.classList.remove('text-gray-700', 'dark:text-gray-300');
        arrow.classList.add('text-gray-400');
    });

    // Show the appropriate arrow for the current column and direction
    const arrowId = `${column}-sort-${currentSortDirection}`;
    const arrow = document.getElementById(arrowId);
    if (arrow) {
        arrow.classList.remove('hidden', 'text-gray-400');
        arrow.classList.add('text-gray-700', 'dark:text-gray-300');
    }
}

function toggleMixedSorting() {
    mixedSorting = !mixedSorting;

    // Update button appearance
    const button = document.getElementById('mixed-sort-toggle');
    const label = document.getElementById('mixed-sort-label');
    const folderIcon = document.getElementById('sort-folder-icon');
    const fileIcon = document.getElementById('sort-file-icon');

    if (mixedSorting) {
        // Mixed sorting enabled - show both icons
        button.title = 'Currently: Mixed sorting (folders and files together)\nClick to switch to folders first';
        label.textContent = 'Mixed';

        // Show both folder and file icons
        folderIcon.classList.remove('invisible');
        fileIcon.classList.remove('invisible');
    } else {
        // Mixed sorting disabled - show only folder icon
        button.title = 'Currently: Folders first sorting\nClick to switch to mixed sorting (folders and files together)';
        label.textContent = 'Folders';

        // Show only folder icon (file icon invisible but still takes space)
        folderIcon.classList.remove('invisible');
        fileIcon.classList.add('invisible');
    }

    // Re-sort if we have a current sort column
    if (currentSortColumn) {
        applySorting();
        updateSortArrows(currentSortColumn);
    }
}

// Keyboard navigation for file table
function handleTableKeyNavigation(event) {
    if (!['ArrowUp', 'ArrowDown', 'ArrowLeft', 'ArrowRight'].includes(event.key)) {
        return;
    }

    const currentElement = document.activeElement;
    const table = document.getElementById('file-explorer-list');
    if (!table || !table.contains(currentElement)) {
        return;
    }

    event.preventDefault();

    if (event.key === 'ArrowDown' || event.key === 'ArrowUp') {
        // Up/Down: Navigate between file rows (focusing on the file name)
        const currentRow = currentElement.closest('tr');
        if (!currentRow) return;

        const allRows = Array.from(table.querySelectorAll('tr'));
        const currentIndex = allRows.indexOf(currentRow);

        if (currentIndex === -1) return;

        let nextIndex;
        if (event.key === 'ArrowDown') {
            nextIndex = currentIndex + 1;
            if (nextIndex >= allRows.length) {
                nextIndex = 0; // Wrap to first row
            }
        } else { // ArrowUp
            nextIndex = currentIndex - 1;
            if (nextIndex < 0) {
                nextIndex = allRows.length - 1; // Wrap to last row
            }
        }

        // Focus on the file name (first focusable element in the row)
        const nextRow = allRows[nextIndex];
        const firstFocusable = nextRow.querySelector('[tabindex="0"]');
        if (firstFocusable) {
            firstFocusable.focus();
        }
    } else if (event.key === 'ArrowLeft' || event.key === 'ArrowRight') {
        // Left/Right: Navigate between elements in the same row
        const currentRow = currentElement.closest('tr');
        if (!currentRow) return;

        const focusableInRow = Array.from(currentRow.querySelectorAll('[tabindex="0"]'));
        const currentIndex = focusableInRow.indexOf(currentElement);

        if (currentIndex === -1) return;

        let nextIndex;
        if (event.key === 'ArrowRight') {
            nextIndex = currentIndex + 1;
            if (nextIndex >= focusableInRow.length) {
                nextIndex = 0; // Wrap to first element in row
            }
        } else { // ArrowLeft
            nextIndex = currentIndex - 1;
            if (nextIndex < 0) {
                nextIndex = focusableInRow.length - 1; // Wrap to last element in row
            }
        }

        focusableInRow[nextIndex].focus();
    }
}

// Add event listener for keyboard navigation
document.addEventListener('keydown', handleTableKeyNavigation);

// Add keyboard support for sort buttons
document.addEventListener('keydown', function (event) {
    const activeElement = document.activeElement;

    if (event.key === 'Enter' || event.key === ' ') {
        // Check if focused element is a sort button
        if (activeElement && activeElement.classList.contains('sort-button')) {
            console.log('Sort button keyboard event detected:', activeElement.id);
            event.preventDefault();
            event.stopPropagation();
            event.stopImmediatePropagation();

            // Extract column name from button id (format: "sort-{columnName}")
            const columnName = activeElement.id.replace('sort-', '');

            if (columnName) {
                console.log('Sorting by column:', columnName);
                sortFiles(columnName);
                updateSortArrows(columnName);
            }

            return false;
        }

        // Check if focused element is the mixed sort toggle
        if (activeElement && activeElement.id === 'mixed-sort-toggle') {
            console.log('Sort switcher keyboard event detected');
            event.preventDefault();
            event.stopPropagation();
            event.stopImmediatePropagation();

            toggleMixedSorting();

            return false;
        }
    }
}, true); // Use capture phase to intercept before other handlers

// Add keyboard shortcut for creating new folder
document.addEventListener('keydown', function (event) {
    // Check if the '+' key is pressed (can be '+' or '=' with shift)
    if ((event.key === '+' || event.key === '=') && !event.ctrlKey && !event.metaKey && !event.altKey) {
        // Don't trigger if user is typing in an input field
        const activeElement = document.activeElement;
        if (activeElement && (activeElement.tagName === 'INPUT' || activeElement.tagName === 'TEXTAREA')) {
            return;
        }

        // Get the new folder button
        const addFolderBtn = document.getElementById('add-folder-btn');
        if (addFolderBtn) {
            event.preventDefault();
            // Click the button to show the input
            addFolderBtn.click();
        }
    }
});

// COLUMN VIEW AUTO-SCROLL
// Automatically scroll column view to show the rightmost (active) column
function scrollColumnViewToRight() {
    const columnViewColumns = document.querySelector('.column-view-columns');
    if (columnViewColumns) {
        // Scroll to the right to show the active column
        columnViewColumns.scrollLeft = columnViewColumns.scrollWidth;
    }
}

// Listen for HTMX content swaps to scroll column view
document.body.addEventListener('htmx:afterSwap', function(event) {
    // Check if we're in column view and the content was swapped
    if (event.detail.target.id === 'file-explorer-view-content') {
        // Small delay to ensure DOM is fully rendered
        requestAnimationFrame(() => {
            scrollColumnViewToRight();
        });
    }
});

// Also scroll on initial page load
document.addEventListener('DOMContentLoaded', function() {
    scrollColumnViewToRight();
});

// NAVIGATION MANAGEMENT

/**
 * Update the back button state based on current path
 */
function updateBackButton() {
    const backBtn = document.getElementById('nav-back-btn');
    if (!backBtn) return;
    
    const currentPath = window.location.pathname;
    const isAtRoot = currentPath === '/files' || currentPath === '/files/';
    
    if (isAtRoot) {
        backBtn.disabled = true;
    } else {
        backBtn.disabled = false;
    }
}

/**
 * Navigate back to previous folder
 */
function navigateBack() {
    const currentPath = window.location.pathname;
    
    // Calculate parent directory
    let parentPath = currentPath.replace(/\/$/, ''); // Remove trailing slash
    const lastSlashIndex = parentPath.lastIndexOf('/');
    parentPath = parentPath.substring(0, lastSlashIndex) || '/files';
    
    // Navigate to parent
    window.history.pushState({}, '', parentPath);
    htmx.ajax('GET', parentPath, {
        target: '#file-explorer-view-content',
        swap: 'innerHTML'
    });
    updateBackButton();
}

// Handle browser back/forward buttons
window.addEventListener('popstate', function(event) {
    // Reload the file explorer content for the current URL
    const currentPath = window.location.pathname;
    htmx.ajax('GET', currentPath, {
        target: '#file-explorer-view-content',
        swap: 'innerHTML'
    });
    updateBackButton();
});

// Update back button on page load
document.addEventListener('DOMContentLoaded', function() {
    updateBackButton();
});

// CONTEXT MENU GLOBAL HANDLERS
// Close context menus when clicking outside
document.addEventListener('click', function(event) {
    // Check if click is outside any context menu
    if (!event.target.closest('.context-menu') && !event.target.closest('.context-menu-trigger') && !event.target.closest('.grid-view-context-trigger') && !event.target.closest('.column-view-context-trigger')) {
        document.querySelectorAll('.context-menu:not(.hidden)').forEach(menu => {
            menu.classList.add('hidden');
        });
    }

    // Clear file selection when clicking on empty space (not on a file node)
    if (!event.target.closest('.file-node') &&
        !event.target.closest('.context-menu') &&
        !event.target.closest('dialog')) {
        clearSelectedFiles();
    }
});
