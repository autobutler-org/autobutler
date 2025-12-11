// Initialize autobutler namespace
window.ab = Object.assign(window.ab || {}, {
    ///////////////
    // Constants //
    ///////////////
    MAX_FILE_NAME_LENGTH: 255,
    DOUBLE_CLICK_DELAY: 300, // ms
    DOUBLE_TAP_DELAY: 300, // ms
    VIEW_STORAGE_KEY: 'fileExplorerView',
    DEVICE_BADGE_STORAGE_KEY: 'showDeviceBadges',

    // File viewer state
    navigationListener: null,
    loadedBook: null,

    // Selection state
    selectedFiles: [],
    clickTimer: null,
    lastSelectedNode: null,

    // Touch state for mobile double-tap detection
    lastTapTime: 0,
    lastTapNode: null,

    // Sorting state
    currentSortColumn: null,
    currentSortDirection: 'asc', // 'asc' or 'desc'
    mixedSorting: false, // false = folders first, true = mixed sorting

    // Device filtering state
    activeDevices: new Set(),

    ///////////////////////
    // General Functions //
    ///////////////////////

    preventDefault: (event) => {
        if (event) {
            event.preventDefault();
            event.stopPropagation();
        }
    },

    debounce: (func, wait) => {
        let timeout;
        return (...args) => {
            clearTimeout(timeout);
            timeout = setTimeout(() => func.apply(this, args), wait);
        };
    },

    ///////////////
    // Functions //
    ///////////////
    getViewPreference: () => {
        return localStorage.getItem(window.ab.VIEW_STORAGE_KEY) || 'list';
    },

    setViewPreference: (view) => {
        localStorage.setItem(window.ab.VIEW_STORAGE_KEY, view);
        // Also set cookie so server can read it on regular page loads
        document.cookie = `fileExplorerView=${view}; path=/; max-age=31536000`; // 1 year
    },

    switchView: (view) => {
        window.ab.setViewPreference(view);

        // Update button states immediately
        window.ab.updateViewButtonStates(view);

        // Use HTMX to reload the content without a full page refresh
        const currentPath = window.location.pathname;
        htmx.ajax('GET', currentPath, {
            target: '#file-explorer-view-content',
            swap: 'innerHTML',
        });
    },

    updateViewButtonStates: (activeView) => {
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
    },

    showFileDetails: (fileName) => {
        alert(fileName);
    },

    toggleFolderInput: (event) => {
        event.preventDefault();
        const folderInput = document.getElementById('folder-input');
        if (!folderInput.classList.toggle('hidden')) {
            folderInput.focus();
        }
    },
    // DEVICE BADGE TOGGLE

    toggleDeviceBadges: (show) => {
        const fileExplorer = document.getElementById('file-explorer');
        if (!fileExplorer) return;

        if (show) {
            fileExplorer.classList.remove('hide-device-badges');
            localStorage.setItem(window.ab.DEVICE_BADGE_STORAGE_KEY, 'true');
        } else {
            fileExplorer.classList.add('hide-device-badges');
            localStorage.setItem(window.ab.DEVICE_BADGE_STORAGE_KEY, 'false');
        }
    },

    // Initialize device badge visibility on page load
    initializeDeviceBadgeToggle: () => {
        const checkbox = document.getElementById('toggle-device-badges');
        if (!checkbox) return;

        // Check if there are multiple managed devices
        const deviceFilterButtons = document.querySelectorAll('.device-filter-button');
        const hasMultipleDevices = deviceFilterButtons.length > 1;

        // Get stored preference, defaulting to "on" if multiple devices, "off" if single device
        const storedPreference = localStorage.getItem(window.ab.DEVICE_BADGE_STORAGE_KEY);
        let showBadges;

        if (storedPreference !== null) {
            showBadges = storedPreference === 'true';
        } else {
            showBadges = hasMultipleDevices;
        }

        checkbox.checked = showBadges;
        window.ab.toggleDeviceBadges(showBadges);
    },

    // Clear file selection when clicking on empty space
    initializeFileSelectionClear: () => {
        document.addEventListener('click', function (event) {
            // Don't clear if clicking on a file node
            if (event.target.closest('.file-node')) {
                return;
            }

            // Don't clear if clicking on the download button
            if (event.target.closest('#file-download-button')) {
                return;
            }

            // Clear selections for any other click
            window.ab.clearSelectedFiles();
        });
    },

    // SELECTION MANAGEMENT (Google Drive style)

    /**
     * Get all file nodes in the current view in DOM order
     */
    getAllFileNodes: () => {
        return Array.from(document.querySelectorAll('.file-node'));
    },

    /**
     * Select a range of file nodes between two nodes (inclusive)
     */
    selectRange: (startNode, endNode) => {
        const allNodes = window.ab.getAllFileNodes();
        const startIndex = allNodes.indexOf(startNode);
        const endIndex = allNodes.indexOf(endNode);

        if (startIndex === -1 || endIndex === -1) return;

        // Determine the range direction
        const minIndex = Math.min(startIndex, endIndex);
        const maxIndex = Math.max(startIndex, endIndex);

        // Select all nodes in the range
        for (let i = minIndex; i <= maxIndex; i++) {
            window.ab.selectFileNode(allNodes[i]);
        }
    },

    /**
     * Clear all selected files and remove visual selection
     */
    clearSelectedFiles: () => {
        document.querySelectorAll('.file-node--selected').forEach((node) => {
            node.classList.remove('file-node--selected');
        });
        window.ab.selectedFiles = [];
        window.ab.updateDownloadButton();
    },

    /**
     * Select a single file node
     */
    selectFileNode: (node) => {
        if (!node) return;

        // Add selection class with temporary logging for debugging
        node.classList.add('file-node--selected');

        // Temporarily log for debugging
        if (window.location.search.includes('debug=1')) {
            console.log('Selected node:', node, 'Classes:', node.className);
        }

        const fileName = node.dataset.name;
        if (fileName && !window.ab.selectedFiles.includes(fileName)) {
            window.ab.selectedFiles.push(fileName);
        }

        // Track this as the last selected node for range selection
        window.ab.lastSelectedNode = node;

        window.ab.updateDownloadButton();
    },

    /**
     * Deselect a single file node
     */
    deselectFileNode: (node) => {
        if (!node) return;
        node.classList.remove('file-node--selected');
        const fileName = node.dataset.name;
        if (fileName) {
            window.ab.selectedFiles = window.ab.selectedFiles.filter((name) => name !== fileName);
        }
        window.ab.updateDownloadButton();
    },

    /**
     * Update the download button state based on selection
     */
    updateDownloadButton: () => {
        const downloadBtn = document.getElementById('file-download-button');
        if (!downloadBtn) return;

        if (window.ab.selectedFiles.length > 0) {
            downloadBtn.disabled = false;
            downloadBtn.classList.remove('btn--disabled');
            downloadBtn.classList.add('btn--secondary');
        } else {
            downloadBtn.disabled = true;
            downloadBtn.classList.remove('btn--secondary');
            downloadBtn.classList.add('btn--disabled');
        }
    },

    /**
     * Handle single click on a file node
     * Single click = select the file (Google Drive style)
     * In column view: single click navigates/opens immediately (Finder style)
     */
    handleFileNodeClick: (event, node) => {
        // Ignore if clicking on context menu trigger
        if (event.target.closest('.context-menu-trigger')) {
            return;
        }

        // Check if we're in column view
        const inColumnView = document.querySelector('.column-view-container') !== null;

        // In column view, single click navigates/opens immediately
        if (inColumnView) {
            window.ab.preventDefault(event);
            const fileType = node.dataset.fileType;

            if (fileType === 'folder') {
                // Navigate to folder
                const contentCell = node.querySelector('[data-href]');
                const href = contentCell?.dataset.href;
                if (href) {
                    htmx.ajax('GET', href, {
                        target: '#file-explorer-view-content',
                        swap: 'innerHTML',
                    }).then(() => {
                        window.history.pushState({}, '', href);
                        window.ab.updateBackButton();
                    });
                }
            } else {
                // Load file preview in preview pane
                const viewerCell = node.querySelector('[data-viewer-path]');
                const viewerPath = viewerCell?.dataset.viewerPath;
                if (viewerPath) {
                    // Clear the preview content completely before loading new content
                    const previewContent = document.getElementById('column-preview-content');
                    if (previewContent) {
                        previewContent.innerHTML = '';
                    }

                    // Load the new preview content
                    htmx.ajax('GET', viewerPath, {
                        target: '#column-preview-content',
                        swap: 'innerHTML',
                    });
                }
            }
            return;
        }

        // For list/grid views, use double-click delay and selection logic
        // Clear any pending double-click timer
        if (window.ab.clickTimer) {
            clearTimeout(window.ab.clickTimer);
            window.ab.clickTimer = null;
        }

        // Wait to see if this becomes a double-click
        window.ab.clickTimer = setTimeout(() => {
            window.ab.clickTimer = null;

            // Single click behavior - toggle selection
            if (event.ctrlKey || event.metaKey) {
                // Ctrl/Cmd+Click: toggle this item's selection
                if (node.classList.contains('file-node--selected')) {
                    window.ab.deselectFileNode(node);
                } else {
                    window.ab.selectFileNode(node);
                }
            } else if (event.shiftKey) {
                // Shift+Click: range selection
                if (window.ab.lastSelectedNode && window.ab.lastSelectedNode !== node) {
                    // Select range from last selected to current
                    window.ab.selectRange(window.ab.lastSelectedNode, node);
                } else {
                    // No previous selection, just select this item
                    window.ab.clearSelectedFiles();
                    window.ab.selectFileNode(node);
                }
            } else {
                // Regular click: select only this item
                window.ab.clearSelectedFiles();
                window.ab.selectFileNode(node);
            }
        }, window.ab.DOUBLE_CLICK_DELAY);
    },

    /**
     * Handle double-click on a file node
     * Double-click = navigate/open the file (Google Drive style)
     */
    handleFileNodeDoubleClick: (event, node) => {
        // Cancel the single-click timer
        if (window.ab.clickTimer) {
            clearTimeout(window.ab.clickTimer);
            window.ab.clickTimer = null;
        }

        window.ab.preventDefault(event);

        const fileType = node.dataset.fileType;

        if (fileType === 'folder') {
            // Navigate to folder - use the stored href
            const contentCell = node.querySelector('[data-href]');
            const href = contentCell?.dataset.href;
            if (href) {
                // Use HTMX for smooth navigation without page reload
                htmx.ajax('GET', href, {
                    target: '#file-explorer-view-content',
                    swap: 'innerHTML',
                }).then(() => {
                    // Update the browser URL after successful navigation
                    window.history.pushState({}, '', href);
                    window.ab.updateBackButton();
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
                        swap: 'innerHTML',
                    });
                }
            }
        }
    },

    /**
     * Handle touch events for mobile double-tap detection
     * On mobile, double-tap opens files (since dblclick doesn't work reliably)
     */
    handleFileNodeTouch: (event, node) => {
        const currentTime = new Date().getTime();
        const tapInterval = currentTime - window.ab.lastTapTime;

        // If this is a second tap on the same node within the delay window
        if (
            window.ab.lastTapNode === node &&
            tapInterval < window.ab.DOUBLE_TAP_DELAY &&
            tapInterval > 0
        ) {
            // Prevent default to avoid zoom on double-tap
            event.preventDefault();

            // Clear the single-tap timer if it's running
            if (window.ab.clickTimer) {
                clearTimeout(window.ab.clickTimer);
                window.ab.clickTimer = null;
            }

            // Trigger the double-click behavior
            window.ab.handleFileNodeDoubleClick(event, node);

            // Reset tap tracking
            window.ab.lastTapTime = 0;
            window.ab.lastTapNode = null;
        } else {
            // Record this tap for potential double-tap
            window.ab.lastTapTime = currentTime;
            window.ab.lastTapNode = node;
        }
    },

    supportsDirectoryUpload: () => {
        const supportsFileSystemAccessAPI = 'getAsFileSystemHandle' in DataTransferItem.prototype;
        const supportsWebkitGetAsEntry = 'webkitGetAsEntry' in DataTransferItem.prototype;
        // NOTE: I have found that none of my browsers support this, and likely is why Google Drive does not support
        // folder upload without a separate input.
        return supportsFileSystemAccessAPI || supportsWebkitGetAsEntry;
    },

    activateDropZone: (event) => {
        window.ab.preventDefault(event);
        const fileUploadArea = document.getElementById('file-upload-area');
        fileUploadArea.classList.add('bg-blue-600');
        fileUploadArea.classList.remove('bg-gray-800');
    },

    deactivateDropZone: (event) => {
        window.ab.preventDefault(event);
        const fileUploadArea = document.getElementById('file-upload-area');
        fileUploadArea.classList.remove('bg-blue-600');
        fileUploadArea.classList.add('bg-gray-800');
    },

    activateDropZoneOnNode: (event) => {
        window.ab.preventDefault(event);
        event.currentTarget.classList.add('bg-blue-600');
    },

    deactivateDropZoneOnNode: (event) => {
        window.ab.preventDefault(event);
        event.currentTarget.classList.remove('bg-blue-600');
    },

    dropOnNode: (event, returnDir) => {
        window.ab.preventDefault(event);
        event.currentTarget.classList.remove('bg-blue-600');
        const li = event.currentTarget.closest('li');
        const dropDir = li.dataset.name;
        console.log(`Drop on node: ${dropDir}`);
        return window.ab.dropFiles(event, `/${dropDir}`, returnDir ? returnDir : '/');
    },

    downloadSelectedFiles: (event, rootDir) => {
        window.ab.preventDefault(event);

        console.log(
            'Download requested. Root dir:',
            rootDir,
            'Selected files:',
            window.ab.selectedFiles
        );

        if (!rootDir) rootDir = '';

        window.ab.selectedFiles.forEach((fileName) => {
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
        window.ab.clearSelectedFiles();
    },

    dropFiles: (event, rootDir, returnDir) => {
        rootDir = rootDir || '';
        returnDir = returnDir || '';
        window.ab.preventDefault(event);
        const files = event.dataTransfer.files;
        if (files.length > 0) {
            const formData = new FormData();
            for (const file of files) {
                formData.append('files', file);
            }
            const uploadForm = document.getElementById('file-upload-form');
            // NOTE: https://flaviocopes.com/htmx-send-files-using-htmxajax-call/
            htmx.ajax('POST', uploadForm.getAttribute('hx-post') + rootDir, {
                values: {
                    files: formData.getAll('files'),
                    returnDir: returnDir,
                },
                source: uploadForm,
            });
        }
    },

    saveAceEditor: (filePath, content) => {
        fetch(`/api/v1/files${filePath}`, {
            method: 'POST',
            headers: {
                'Content-Type': 'text/plain',
            },
            body: content,
        })
            .then((response) => {
                if (response.ok) {
                    toastr.success('File saved successfully');
                    console.log('File saved successfully');
                } else {
                    return response.text().then((text) => {
                        toastr.error('Error saving file: ' + (text || response.statusText));
                        console.error('Error saving file:', response.statusText);
                    });
                }
            })
            .catch((error) => {
                console.error('Error saving file:', error);
                toastr.error('Error saving file: ' + error.message);
            });
    },

    downloadFile: (filePath) => {
        const link = document.createElement('a');
        link.href = `/api/v1/files${filePath}`;
        link.download = filePath.split('/').pop();
        document.body.appendChild(link);
        link.click();
        document.body.removeChild(link);
    },

    moveFile: (event, rootDir, fileName) => {
        window.ab.preventDefault(event);
        while (rootDir && rootDir[0] == '/') {
            rootDir = rootDir.slice(1);
        }
        const filePath = `${rootDir}/${fileName}`;

        // Open the rename dialog
        window.ab.openRenameDialog(filePath);
    },

    newFile: (event, rootDir) => {
        window.ab.preventDefault(event);
        const fileName = prompt('Enter the new file name (including extension):');
        if (fileName) {
            if (fileName.length > window.ab.MAX_FILE_NAME_LENGTH) {
                alert(`File name must be ${window.ab.MAX_FILE_NAME_LENGTH} characters or less`);
                return;
            }
            const uploadForm = document.getElementById('file-upload-form');
            const formData = new FormData();
            // NOTE: Creating an empty file
            formData.append('files', new Blob([''], { type: 'text/plain' }), fileName);
            htmx.ajax('POST', uploadForm.getAttribute('hx-post') + rootDir, {
                values: {
                    files: formData.getAll('files'),
                    returnDir: rootDir,
                },
                source: uploadForm,
            });
        }
    },

    showFolderDetails: (event) => {
        window.ab.preventDefault(event);
        alert('Folder details to be implemented.');
    },

    navigateToParentAndPreview: (event, parentPath, previewPath) => {
        window.ab.preventDefault(event);
        // Use HTMX to navigate to parent (removes child columns) without full page reload
        htmx.ajax('GET', parentPath, {
            target: '#file-explorer-view-content',
            swap: 'innerHTML',
        }).then(function () {
            // After the file explorer updates, load the preview
            htmx.ajax('GET', previewPath, {
                target: '#column-preview-content',
                swap: 'innerHTML',
            });
        });
        // Update the URL
        history.pushState({}, '', parentPath);
    },

    // SORTING

    sortFiles: (column) => {
        if (window.ab.currentSortColumn === column) {
            window.ab.currentSortDirection =
                window.ab.currentSortDirection === 'asc' ? 'desc' : 'asc';
        } else {
            window.ab.currentSortDirection = 'asc';
        }
        window.ab.currentSortColumn = column;

        window.ab.applySorting();
    },

    applySorting: () => {
        const column = window.ab.currentSortColumn;
        if (!column) return;

        const tbody = document.getElementById('file-explorer-list');
        const rows = Array.from(tbody.querySelectorAll('tr'));

        // Separate the spacer row (last row with "Drop files here...")
        const spacerRow = rows.find((row) => row.querySelector('.spacer'));
        const fileRows = rows.filter((row) => row !== spacerRow);

        fileRows.sort((a, b) => {
            let aValue, bValue;

            if (column === 'name') {
                aValue = a.dataset.name || '';
                bValue = b.dataset.name || '';

                // Sort folders first, then files (unless mixed sorting is enabled)
                if (!window.ab.mixedSorting) {
                    const aIsFolder = a.querySelector('td:first-child a[href]') !== null;
                    const bIsFolder = b.querySelector('td:first-child a[href]') !== null;

                    if (aIsFolder && !bIsFolder) return -1;
                    if (!aIsFolder && bIsFolder) return 1;
                }

                // Sort alphabetically
                return window.ab.currentSortDirection === 'asc'
                    ? aValue.localeCompare(bValue, undefined, {
                          numeric: true,
                          sensitivity: 'base',
                      })
                    : bValue.localeCompare(aValue, undefined, {
                          numeric: true,
                          sensitivity: 'base',
                      });
            } else if (column === 'size') {
                // Sort folders first, then files (unless mixed sorting is enabled)
                if (!window.ab.mixedSorting) {
                    const aIsFolder = a.querySelector('td:first-child a[href]') !== null;
                    const bIsFolder = b.querySelector('td:first-child a[href]') !== null;

                    if (aIsFolder && !bIsFolder) return -1;
                    if (!aIsFolder && bIsFolder) return 1;
                }

                // Extract size text and convert to bytes for comparison
                const aSizeText = a.querySelector('td:nth-child(2)')?.textContent?.trim() || '0 B';
                const bSizeText = b.querySelector('td:nth-child(2)')?.textContent?.trim() || '0 B';

                aValue = window.ab.parseSize(aSizeText);
                bValue = window.ab.parseSize(bSizeText);

                return window.ab.currentSortDirection === 'asc' ? aValue - bValue : bValue - aValue;
            }

            return 0;
        });

        // Clear tbody and re-append sorted rows
        tbody.innerHTML = '';
        fileRows.forEach((row) => tbody.appendChild(row));

        // Add spacer row back at the end
        if (spacerRow) {
            tbody.appendChild(spacerRow);
        }
    },

    parseSize: (sizeText) => {
        const units = {
            B: 1,
            KB: 1024,
            MB: 1024 * 1024,
            GB: 1024 * 1024 * 1024,
            TB: 1024 * 1024 * 1024 * 1024,
        };
        const match = sizeText.match(/^([\d.]+)\s*([A-Z]+)$/);
        if (!match) return 0;

        const value = parseFloat(match[1]);
        const unit = match[2];
        return value * (units[unit] || 1);
    },

    updateSortArrows: (column) => {
        // Hide all arrows first
        const allArrows = document.querySelectorAll('[id$="-sort-asc"], [id$="-sort-desc"]');
        allArrows.forEach((arrow) => {
            arrow.classList.add('hidden');
            arrow.classList.remove('text-gray-700', 'dark:text-gray-300');
            arrow.classList.add('text-gray-400');
        });

        // Show the appropriate arrow for the current column and direction
        const arrowId = `${column}-sort-${window.ab.currentSortDirection}`;
        const arrow = document.getElementById(arrowId);
        if (arrow) {
            arrow.classList.remove('hidden', 'text-gray-400');
            arrow.classList.add('text-gray-700', 'dark:text-gray-300');
        }
    },

    toggleMixedSorting: () => {
        window.ab.mixedSorting = !window.ab.mixedSorting;

        // Update button appearance
        const button = document.getElementById('mixed-sort-toggle');
        const label = document.getElementById('mixed-sort-label');
        const folderIcon = document.getElementById('sort-folder-icon');
        const fileIcon = document.getElementById('sort-file-icon');

        if (window.ab.mixedSorting) {
            // Mixed sorting enabled - show both icons
            button.title =
                'Currently: Mixed sorting (folders and files together)\nClick to switch to folders first';
            label.textContent = 'Mixed';

            // Show both folder and file icons
            folderIcon.classList.remove('invisible');
            fileIcon.classList.remove('invisible');
        } else {
            // Mixed sorting disabled - show only folder icon
            button.title =
                'Currently: Folders first sorting\nClick to switch to mixed sorting (folders and files together)';
            label.textContent = 'Folders';

            // Show only folder icon (file icon invisible but still takes space)
            folderIcon.classList.remove('invisible');
            fileIcon.classList.add('invisible');
        }

        // Re-sort if we have a current sort column
        if (window.ab.currentSortColumn) {
            window.ab.applySorting();
            window.ab.updateSortArrows(window.ab.currentSortColumn);
        }
    },

    // Keyboard navigation for file table
    handleTableKeyNavigation: (event) => {
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
            } else {
                // ArrowUp
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
            } else {
                // ArrowLeft
                nextIndex = currentIndex - 1;
                if (nextIndex < 0) {
                    nextIndex = focusableInRow.length - 1; // Wrap to last element in row
                }
            }

            focusableInRow[nextIndex].focus();
        }
    },

    // NAVIGATION MANAGEMENT

    // Automatically scroll column view to show the rightmost (active) column
    scrollColumnViewToRight: () => {
        const columnViewColumns = document.querySelector('.column-view-columns');
        if (columnViewColumns) {
            // Scroll to the right to show the active column
            columnViewColumns.scrollLeft = columnViewColumns.scrollWidth;
        }
    },

    /**
     * Update the back button state based on current path
     */
    updateBackButton: () => {
        const backBtn = document.getElementById('nav-back-btn');
        if (!backBtn) return;

        const currentPath = window.location.pathname;
        const isAtRoot = currentPath === '/files' || currentPath === '/files/';

        if (isAtRoot) {
            backBtn.disabled = true;
        } else {
            backBtn.disabled = false;
        }
    },

    /**
     * Navigate back to previous folder
     */
    navigateBack: () => {
        const currentPath = window.location.pathname;

        // Calculate parent directory
        let parentPath = currentPath.replace(/\/$/, ''); // Remove trailing slash
        const lastSlashIndex = parentPath.lastIndexOf('/');
        parentPath = parentPath.substring(0, lastSlashIndex) || '/files';

        // Navigate to parent
        window.history.pushState({}, '', parentPath);
        htmx.ajax('GET', parentPath, {
            target: '#file-explorer-view-content',
            swap: 'innerHTML',
        });
        window.ab.updateBackButton();
    },

    // DEVICE FILTERING

    // Initialize active devices on page load
    initializeDeviceFilter: () => {
        const deviceButtons = document.querySelectorAll('.device-filter-button');
        deviceButtons.forEach((button) => {
            const deviceName = button.getAttribute('data-device-name');
            if (button.classList.contains('device-filter-button--active')) {
                window.ab.activeDevices.add(deviceName);
            }
        });
        window.ab.applyDeviceFilter();
    },

    // Toggle device filter when button is clicked
    toggleDeviceFilter: (button) => {
        const deviceName = button.getAttribute('data-device-name');

        // Toggle active state
        button.classList.toggle('device-filter-button--active');

        if (window.ab.activeDevices.has(deviceName)) {
            window.ab.activeDevices.delete(deviceName);
        } else {
            window.ab.activeDevices.add(deviceName);
        }

        window.ab.applyDeviceFilter();
    },

    // Apply the current device filter to all file nodes
    applyDeviceFilter: () => {
        // Get all file nodes
        const fileNodes = document.querySelectorAll('.file-node');

        // If no devices are being filtered (no filter buttons or all inactive), show everything
        if (window.ab.activeDevices.size === 0) {
            fileNodes.forEach((node) => {
                node.setAttribute('data-device-filtered', 'false');
                node.style.display = '';
            });
            return;
        }

        fileNodes.forEach((node) => {
            // Find device badge in this node
            const deviceBadge = node.querySelector('.device-badge-name');

            if (!deviceBadge) {
                // No device badge means it's from the default/primary device
                // Show it when filters are active (for backwards compatibility)
                node.setAttribute('data-device-filtered', 'false');
                node.style.display = '';
            } else {
                const deviceName = deviceBadge.textContent.trim();

                // Show if device is in active set
                if (window.ab.activeDevices.has(deviceName)) {
                    node.setAttribute('data-device-filtered', 'false');
                    node.style.display = '';
                } else {
                    node.setAttribute('data-device-filtered', 'true');
                    node.style.display = 'none';
                }
            }
        });
    },
});

// Initialize - sync localStorage to cookie on page load and update button states
document.addEventListener('DOMContentLoaded', function () {
    const view = window.ab.getViewPreference();
    window.ab.setViewPreference(view); // Ensures cookie is set
    window.ab.updateViewButtonStates(view); // Ensure button states match the active view
});

// Send view preference in all HTMX requests via custom header
document.body.addEventListener('htmx:configRequest', function (event) {
    const view = window.ab.getViewPreference();
    event.detail.headers['X-File-Explorer-View'] = view;
});

// Add event listener for keyboard navigation
document.addEventListener('keydown', window.ab.handleTableKeyNavigation);

// Add keyboard support for sort buttons
document.addEventListener(
    'keydown',
    function (event) {
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
                    window.ab.sortFiles(columnName);
                    window.ab.updateSortArrows(columnName);
                }

                return false;
            }

            // Check if focused element is the mixed sort toggle
            if (activeElement && activeElement.id === 'mixed-sort-toggle') {
                console.log('Sort switcher keyboard event detected');
                event.preventDefault();
                event.stopPropagation();
                event.stopImmediatePropagation();

                window.ab.toggleMixedSorting();

                return false;
            }
        }
    },
    true
); // Use capture phase to intercept before other handlers

// Add keyboard shortcut for creating new folder
document.addEventListener('keydown', function (event) {
    // Check if the '+' key is pressed (can be '+' or '=' with shift)
    if (
        (event.key === '+' || event.key === '=') &&
        !event.ctrlKey &&
        !event.metaKey &&
        !event.altKey
    ) {
        // Don't trigger if user is typing in an input field
        const activeElement = document.activeElement;
        if (
            activeElement &&
            (activeElement.tagName === 'INPUT' || activeElement.tagName === 'TEXTAREA')
        ) {
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

// Listen for HTMX content swaps to scroll column view
document.body.addEventListener('htmx:afterSwap', function (event) {
    // Check if we're in column view and the content was swapped
    if (event.detail.target.id === 'file-explorer-view-content') {
        // Small delay to ensure DOM is fully rendered
        requestAnimationFrame(() => {
            window.ab.scrollColumnViewToRight();
        });
    }
});

// Also scroll on initial page load
document.addEventListener('DOMContentLoaded', window.ab.scrollColumnViewToRight);

// Handle browser back/forward buttons
window.addEventListener('popstate', function () {
    // Reload the file explorer content for the current URL
    const currentPath = window.location.pathname;
    htmx.ajax('GET', currentPath, {
        target: '#file-explorer-view-content',
        swap: 'innerHTML',
    });
    window.ab.updateBackButton();
});

// Update back button on page load
document.addEventListener('DOMContentLoaded', function () {
    window.ab.updateBackButton();
});

// Initialize on page load and after HTMX swaps
document.addEventListener('DOMContentLoaded', window.ab.initializeDeviceFilter);
document.body.addEventListener('htmx:afterSwap', window.ab.initializeDeviceFilter);

document.addEventListener('DOMContentLoaded', window.ab.initializeDeviceBadgeToggle);
document.body.addEventListener('htmx:afterSwap', window.ab.initializeDeviceBadgeToggle);

document.addEventListener('DOMContentLoaded', window.ab.initializeFileSelectionClear);
