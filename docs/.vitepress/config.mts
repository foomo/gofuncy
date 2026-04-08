import { defineConfig } from 'vitepress'

// https://vitepress.dev/reference/site-config
export default defineConfig({
	title: 'gofuncy',
	description: 'Structured concurrency primitives for Go',
	lang: "en-US",
	lastUpdated: true,
	appearance: "force-dark",
	ignoreDeadLinks: true,
	base: '/gofuncy/',
	sitemap: {
		hostname: 'https://foomo.github.io/gofuncy',
	},
	themeConfig: {
		// https://vitepress.dev/reference/default-theme-config
		logo: '/logo.png',
		outline: [2, 4],
		nav: [
			{ text: 'Guide', link: '/guide/introduction' },
			{ text: 'API Reference', link: '/api/go' },
			{ text: 'Examples', link: '/examples/basic' },
		],
		sidebar: [
			{
				text: 'Guide',
				items: [
					{ text: 'Introduction', link: '/guide/introduction' },
					{ text: 'Getting Started', link: '/guide/getting-started' },
					{ text: 'Core Concepts', link: '/guide/concepts' },
				],
			},
			{
				text: 'API Reference',
				items: [
					{ text: 'Go', link: '/api/go' },
					{ text: 'Group', link: '/api/group' },
					{ text: 'ForEach', link: '/api/foreach' },
					{ text: 'Map', link: '/api/map' },
					{ text: 'Options', link: '/api/options' },
				],
			},
			{
				text: 'Examples',
				items: [
					{ text: 'Basic', link: '/examples/basic' },
					{ text: 'Advanced', link: '/examples/advanced' },
					{ text: 'Patterns', link: '/examples/patterns' },
				],
			},
			{
				text: 'Contributing',
				collapsed: true,
				items: [
					{
						text: "Guideline",
						link: '/CONTRIBUTING.md',
					},
					{
						text: "Code of conduct",
						link: '/CODE_OF_CONDUCT.md',
					},
					{
						text: "Security guidelines",
						link: '/SECURITY.md',
					},
				],
			},
		],
		socialLinks: [
			{ icon: 'github', link: 'https://github.com/foomo/gofuncy' },
		],
		editLink: {
			pattern: 'https://github.com/foomo/gofuncy/edit/main/docs/:path',
		},
		search: {
			provider: 'local',
		},
		footer: {
			message: 'Made with ♥ <a href="https://www.foomo.org">foomo</a> by <a href="https://www.bestbytes.com">bestbytes</a>',
		},
	},
	markdown: {
		// https://github.com/vuejs/vitepress/discussions/3724
		theme: {
			dark: 'github-dark',
			light: 'github-light',
		}
	},
	head: [
		['meta', { name: 'theme-color', content: '#ffffff' }],
		['link', { rel: 'icon', href: '/logo.png' }],
		['meta', { name: 'author', content: 'foomo by bestbytes' }],
		// OpenGraph
		['meta', { property: 'og:title', content: 'foomo/gofuncy' }],
		[
			'meta',
			{
				property: 'og:image',
				content: 'https://github.com/foomo/gofuncy/blob/main/docs/public/banner.png?raw=true',
			},
		],
		[
			'meta',
			{
				property: 'og:description',
				content: 'Structured concurrency primitives for Go',
			},
		],
		['meta', { name: 'twitter:card', content: 'summary_large_image' }],
		[
			'meta',
			{
				name: 'twitter:image',
				content: 'https://github.com/foomo/gofuncy/blob/main/docs/public/banner.png?raw=true',
			},
		],
		[
			'meta', { name: 'viewport', content: 'width=device-width, initial-scale=1.0, viewport-fit=cover',
			},
		],
	]
})
