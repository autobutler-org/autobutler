import { ref, readonly } from 'vue';

// Types
interface MarkdownSection {
  name: string;
  file: string;
}

interface MarkdownCache {
  [key: string]: string;
}

// State
const markdownCache = ref<MarkdownCache>({});
const isLoading = ref(false);
const currentContent = ref('');
const currentSection = ref('');

// Pure functions for markdown parsing
const escapeHtml = (text: string): string => 
  text.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');

const unescapeHtml = (text: string): string =>
  text.replace(/&amp;/g, '&').replace(/&lt;/g, '<').replace(/&gt;/g, '>');

const generateSlug = (text: string): string =>
  text
    .toLowerCase()
    .trim()
    .replace(/[^\w\s-]/g, '')
    .replace(/\s+/g, '-')
    .replace(/-+/g, '-')
    .replace(/^-|-$/g, '');

// Markdown parsing functions
const parseCodeBlocks = (html: string): string =>
  html.replace(/```(\w+)?\n?([\s\S]*?)```/g, (match, lang, code) => {
    const language = lang || '';
    const cleanCode = unescapeHtml(code.trim());
    return `<pre><code class="language-${language}">${cleanCode}</code></pre>`;
  });

const parseInlineCode = (html: string): string =>
  html.replace(/`([^`\n]+)`/g, '<code>$1</code>');

const parseHeaders = (html: string): string => {
  const headerReplacements = [
    [/^### (.*)$/gim, '<h3>$1</h3>'],
    [/^## (.*)$/gim, '<h2>$1</h2>'],
    [/^# (.*)$/gim, '<h1>$1</h1>']
  ] as const;

  return headerReplacements.reduce((acc, [pattern, replacement]) => 
    acc.replace(pattern, replacement), html);
};

const parseBoldAndItalic = (html: string): string => {
  const formatted = html.replace(/\*\*(.*?)\*\*/g, '<strong>$1</strong>');
  
  return formatted.replace(/\*([^*\n]+)\*/g, (match, content) => {
    if (match.startsWith('**') || match.endsWith('**')) {
      return match;
    }
    return `<em>${content}</em>`;
  });
};

const parseLinks = (html: string): string =>
  html.replace(/\[([^\]]+)\]\(([^)]+)\)/g, '<a href="$2">$1</a>');

const parseLists = (html: string): string => {
  const withListItems = html
    .replace(/^[-*+] (.+)$/gm, '<li>$1</li>')
    .replace(/^\d+\. (.+)$/gm, '<li>$1</li>');
    
  return withListItems.replace(/(<li>.*<\/li>\s*)+/gs, '<ul>$&</ul>');
};

const parseBlockquotes = (html: string): string =>
  html.replace(/^> (.+)$/gm, '<blockquote><p>$1</p></blockquote>');

const parseHorizontalRules = (html: string): string =>
  html.replace(/^(---|\*\*\*)$/gm, '<hr>');

const wrapInParagraphs = (html: string): string => {
  const withParaBreaks = html.replace(/\n\s*\n/g, '</p><p>');
  return `<p>${withParaBreaks}</p>`;
};

const cleanupParagraphs = (html: string): string => {
  const blockElements = ['h1', 'h2', 'h3', 'h4', 'h5', 'h6', 'pre', 'ul', 'ol', 'table', 'blockquote', 'hr'];
  
  const cleanedHtml = blockElements.reduce((acc, tag) => {
    const withoutEmpty = acc.replace(/<p><\/p>/g, '');
    const withoutAroundOpening = withoutEmpty.replace(new RegExp(`<p>(<${tag}[^>]*>)`, 'g'), '$1');
    return withoutAroundOpening.replace(new RegExp(`(</${tag}>)</p>`, 'g'), '$1');
  }, html);

  return cleanedHtml
    .replace(/\n\s*\n/g, '\n')
    .replace(/^\s+|\s+$/g, '');
};

const restoreCodeEntities = (html: string): string =>
  html.replace(/<code[^>]*>([^<]*)<\/code>/g, (match) => 
    match.replace(/&amp;/g, '&').replace(/&lt;/g, '<').replace(/&gt;/g, '>'));

// Main markdown parsing pipeline
const parseMarkdown = (markdown: string): string => {
  const pipeline = [
    escapeHtml,
    parseCodeBlocks,
    parseInlineCode,
    parseHeaders,
    parseBoldAndItalic,
    parseLinks,
    parseLists,
    parseBlockquotes,
    parseHorizontalRules,
    wrapInParagraphs,
    cleanupParagraphs,
    restoreCodeEntities
  ];

  return pipeline.reduce((content, fn) => fn(content), markdown);
};

// Generate IDs for headers for navigation
const addHeaderIds = (html: string): string =>
  html.replace(/<h([1-6])>([^<]+)<\/h[1-6]>/g, (match, level, text) => {
    const id = generateSlug(text);
    return `<h${level} id="${id}">${text}</h${level}>`;
  });

// Cache management
const getCachedContent = (sectionName: string): string | null =>
  markdownCache.value[sectionName] || null;

const setCachedContent = (sectionName: string, content: string): void => {
  markdownCache.value[sectionName] = content;
};

// Content loading
const fetchMarkdownContent = async (filename: string): Promise<string> => {
  const response = await fetch(`/docs/${filename}`);
  if (!response.ok) {
    throw new Error(`Failed to fetch ${filename}: ${response.status}`);
  }
  return response.text();
};

const generateFallbackContent = (sectionName: string): string => {
  const fallbackMap: Record<string, string> = {
    'Getting Started': `
      <h1>Getting Started</h1>
      <p>Welcome to AutoButler! This guide will help you get up and running quickly.</p>
      <h2>Prerequisites</h2>
      <p>Before you begin, make sure you have Node.js version 16 or higher installed.</p>
    `,
    'Quick Start': `
      <h1>Quick Start</h1>
      <p>Get AutoButler running in under 5 minutes!</p>
      <h2>Installation</h2>
      <pre><code>npm install @autobutler/cli -g
autobutler init my-project</code></pre>
    `,
    'Installation': `
      <h1>Installation</h1>
      <p>Detailed installation instructions for AutoButler.</p>
      <h2>System Requirements</h2>
      <p>AutoButler requires Node.js 16+ and supports all major operating systems.</p>
    `,
    'Configuration': `
      <h1>Configuration</h1>
      <p>Configure AutoButler to work exactly how you need it.</p>
      <h2>Basic Configuration</h2>
      <pre><code>version: "1.0"
project: "my-project"
environment: "development"</code></pre>
    `,
    'API Reference': `
      <h1>API Reference</h1>
      <p>Complete API documentation for AutoButler.</p>
      <h2>Core Classes</h2>
      <p>The main AutoButler class provides the primary interface for automation tasks.</p>
    `,
    'Examples': `
      <h1>Examples</h1>
      <p>Real-world examples to help you get the most out of AutoButler.</p>
      <h2>Basic Data Fetching</h2>
      <p>Learn how to fetch and process data with AutoButler.</p>
    `
  };

  return fallbackMap[sectionName] || `
    <h1>${sectionName}</h1>
    <p>Documentation content for ${sectionName} is currently being loaded.</p>
    <p>Please check back later or contact support if this issue persists.</p>
  `;
};

// Main composable function
export const useMarkdown = () => {
  const loadSection = async (section: MarkdownSection): Promise<void> => {
    isLoading.value = true;
    currentSection.value = section.name;

    try {
      // Check cache first
      const cached = getCachedContent(section.name);
      if (cached) {
        currentContent.value = cached;
        return;
      }

      // Fetch and parse markdown
      const markdownContent = await fetchMarkdownContent(section.file);
      const htmlContent = parseMarkdown(markdownContent);
      const contentWithIds = addHeaderIds(htmlContent);

      // Cache and set content
      setCachedContent(section.name, contentWithIds);
      currentContent.value = contentWithIds;

    } catch (error) {
      console.error(`Error loading section ${section.name}:`, error);
      const fallbackContent = generateFallbackContent(section.name);
      currentContent.value = fallbackContent;
    } finally {
      isLoading.value = false;
    }
  };

  const clearCache = (): void => {
    markdownCache.value = {};
  };

  return {
    // State (readonly)
    isLoading: readonly(isLoading),
    currentContent: readonly(currentContent),
    currentSection: readonly(currentSection),
    
    // Actions
    loadSection,
    clearCache,
    
    // Utilities
    generateSlug
  };
}; 