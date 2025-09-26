var contextMenuPosition = {
    x: null,
    y: null,
};
// NOTE: Must be global so that it can be removed when the file viewer is closed.
var navigationListener = null;
var loadedBook = null;

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
    const contextMenu = openContextMenu(event, parentNode);
    contextMenu.style.left = `${event.clientX}px`;
    contextMenu.style.top = `${event.clientY}px`;
}

function toggleFolderInput(event) {
    event.preventDefault();
    const folderInput = document.getElementById('folder-input');
    if (!folderInput.classList.toggle('hidden')) {
        folderInput.focus();
    }
}

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
    selectedFiles.forEach(fileName => {
        const link = document.createElement('a');
        while (fileName.endsWith('/')) {
            fileName = fileName.slice(0, -1);
        }
        link.href = `/api/v1/files${rootDir}/${fileName}`;
        link.download = fileName;
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
    fetch(`/api/v1/files${filePath}`, {
        method: 'PUT',
        headers: {
            'Content-Type': 'application/json'
        },
        body: JSON.stringify(delta)
    }).catch(error => {
        console.error('Error updating file:', error);
    });
}

function newFile(event, rootDir) {
    preventDefault(event);
    const fileName = prompt("Enter the new file name (including extension):");
    if (fileName) {
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

// SELECTO

var SELECTABLE_TARGETS = ['.file-node'];

function clearSelectedFiles() {
    setSelectedFiles([]);
    const fileNodes = document.querySelectorAll('.file-node');
    fileNodes.forEach(node => {
        node.classList.remove(...selectoClasses);
    });
}

function setSelectedFiles(fileNames) {
    selectedFiles = fileNames;
    const hasSelectedFiles = selectedFiles.length > 0;
    document.getElementById('file-delete-button').disabled = !hasSelectedFiles;
    document.getElementById('file-download-button').disabled = !hasSelectedFiles;
}

// SELECTO
var selectoClasses = ["bg-gray-100", "dark:bg-gray-800"];
var selectedFiles = [];
var selecto = new Selecto({
    // The container to add a selection element
    container: document.body,
    // Selecto's root container (No transformed container. (default: null)
    rootContainer: null,
    // The area to drag selection element (default: container)
    dragContainer: document.getElementById('file-explorer-selectable'),
    // Targets to select. You can register a queryselector or an Element.
    selectableTargets: SELECTABLE_TARGETS,
    // Whether to select by click (default: true)
    selectByClick: false,
    // Whether to select from the target inside (default: true)
    selectFromInside: true,
    // After the select, whether to select the next target with the selected target (deselected if the target is selected again).
    continueSelect: false,
    // Determines which key to continue selecting the next target via keydown and keyup.
    toggleContinueSelect: "shift",
    // The container for keydown and keyup events
    keyContainer: window,
    // The rate at which the target overlaps the drag area to be selected. (default: 100)
    // NOTE: Percentage of target area that must be enclosed by selection box to be selected.
    hitRate: 1,
});
selecto.on("select", e => {
    e.added.forEach(el => {
        el.classList.add(...selectoClasses);
    });
    e.removed.forEach(el => {
        el.classList.remove(...selectoClasses);
    });
});
selecto.on('selectEnd', e => {
    setSelectedFiles(e.selected.map(el => el.dataset.name));
});
