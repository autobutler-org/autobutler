function initializeTextEditor(filePath) {
	if (typeof ace === 'undefined') {
		console.error('ACE editor not loaded');
		return;
	}

	var editor = ace.edit("editor");
	editor.setTheme("ace/theme/monokai");

	// Determine ACE mode based on file extension
	function getAceModeFromExtension(filePath) {
		var ext = filePath.split('.').pop().toLowerCase();
		var modeMap = {
			// Web languages
			'js': 'javascript',
			'jsx': 'javascript',
			'ts': 'typescript',
			'tsx': 'typescript',
			'json': 'json',
			'html': 'html',
			'htm': 'html',
			'xml': 'xml',
			'css': 'css',
			'scss': 'scss',
			'sass': 'sass',
			'less': 'less',

			// Programming languages
			'py': 'python',
			'rb': 'ruby',
			'php': 'php',
			'java': 'java',
			'c': 'c_cpp',
			'cpp': 'c_cpp',
			'cc': 'c_cpp',
			'h': 'c_cpp',
			'hpp': 'c_cpp',
			'cs': 'csharp',
			'go': 'golang',
			'rs': 'rust',
			'swift': 'swift',
			'kt': 'kotlin',
			'scala': 'scala',
			'r': 'r',
			'lua': 'lua',
			'pl': 'perl',

			// Shell/Config
			'sh': 'sh',
			'bash': 'sh',
			'zsh': 'sh',
			'fish': 'sh',
			'yaml': 'yaml',
			'yml': 'yaml',
			'toml': 'toml',
			'ini': 'ini',
			'conf': 'ini',
			'env': 'ini',

			// Markup
			'md': 'markdown',
			'markdown': 'markdown',
			'rst': 'rst',
			'tex': 'latex',

			// Data
			'sql': 'sql',
			'csv': 'text',

			// Other
			'dockerfile': 'dockerfile',
			'makefile': 'makefile'
		};

		return modeMap[ext] || 'text';
	}

	var mode = getAceModeFromExtension(filePath);
	editor.getSession().setMode('ace/mode/' + mode);

	// Force editor to resize and render properly
	editor.resize();
	editor.renderer.updateFull();

	// Debounced save function
	let saveTimeout;
	editor.getSession().on('change', function() {
		clearTimeout(saveTimeout);
		saveTimeout = setTimeout(function() {
			var content = editor.getValue();
			var blob = new Blob([content], { type: 'text/plain' });
			var formData = new FormData();

			// Extract filename from path for the blob
			var fileName = filePath.split('/').pop();
			formData.append('files', blob, fileName);

			// Extract directory from full path
			var dirPath = filePath.substring(0, filePath.lastIndexOf('/'));

			// POST to update the file
			fetch('/api/v1/files' + dirPath, {
				method: 'POST',
				body: formData
			}).catch(function(error) {
				console.error('Error saving file:', error);
			});
		}, 1000); // Save after 1 second of inactivity
	});
}
