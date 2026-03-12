// @ts-check
import { defineConfig } from 'astro/config';
import starlight from '@astrojs/starlight';
import starlightLlmsTxt from 'starlight-llms-txt';

export default defineConfig({
	site: 'https://zach-snell.github.io',
	base: '/ctk',
	integrations: [
		starlight({
			title: 'ctk',
			description: 'A dual-mode Go CLI & MCP Server for Confluence Cloud',
			plugins: [
				starlightLlmsTxt({
					projectName: 'ctk (Confluence Toolkit)',
					description: 'A dual-mode Go CLI and MCP Server for Confluence Cloud. Provides 8 MCP tools with 30+ actions for spaces, pages, search, labels, folders, comments, attachments, and users. Features XHTML storage format to markdown conversion, markdown-to-storage conversion, page version diffing, inline comment reading, write gating, response flattening, and rate limiting. The only Confluence MCP with folder support and page diff.',
					customSets: [
						{
							label: 'MCP Tools',
							description: 'All MCP tool reference documentation',
							paths: ['mcp/**'],
						},
						{
							label: 'CLI',
							description: 'CLI command reference',
							paths: ['cli/**'],
						},
					],
				}),
			],
			social: [
				{ icon: 'github', label: 'GitHub', href: 'https://github.com/zach-snell/ctk' },
			],
			editLink: {
				baseUrl: 'https://github.com/zach-snell/ctk/edit/main/docs/',
			},
			customCss: ['./src/styles/custom.css'],
			sidebar: [
				{
					label: 'Getting Started',
					items: [
						{ label: 'Introduction', slug: 'getting-started/introduction' },
						{ label: 'Installation', slug: 'getting-started/installation' },
						{ label: 'Configuration', slug: 'getting-started/configuration' },
						{ label: 'Quick Start', slug: 'getting-started/quickstart' },
					],
				},
				{
					label: 'CLI Commands',
					items: [
						{ label: 'Overview', slug: 'cli/overview' },
						{ label: 'ctk spaces', slug: 'cli/spaces' },
						{ label: 'ctk pages', slug: 'cli/pages' },
						{ label: 'ctk folders', slug: 'cli/folders' },
						{ label: 'ctk search', slug: 'cli/search' },
					],
				},
				{
					label: 'MCP Tool Reference',
					items: [
						{ label: 'Overview', slug: 'mcp/overview' },
						{ label: 'manage_spaces', slug: 'mcp/manage-spaces' },
						{ label: 'manage_pages', slug: 'mcp/manage-pages' },
						{ label: 'manage_search', slug: 'mcp/manage-search' },
						{ label: 'manage_labels', slug: 'mcp/manage-labels' },
						{ label: 'manage_folders', slug: 'mcp/manage-folders' },
						{ label: 'manage_comments', slug: 'mcp/manage-comments' },
					{ label: 'manage_attachments', slug: 'mcp/manage-attachments' },
					{ label: 'manage_users', slug: 'mcp/manage-users' },
				],
				},
				{
					label: 'Guides',
					items: [
						{ label: 'Usage Examples', slug: 'guides/examples' },
						{ label: 'Storage Format', slug: 'guides/storage-format' },
					],
				},
				{
					label: 'Advanced',
					items: [
						{ label: 'Architecture', slug: 'advanced/architecture' },
						{ label: 'Security', slug: 'advanced/security' },
						{ label: 'Development', slug: 'advanced/development' },
					],
				},
			],
		}),
	],
});
