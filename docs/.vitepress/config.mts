import { defineConfig } from 'vitepress'

// https://vitepress.dev/reference/site-config
export default defineConfig({
	title: 'gofuncy',
	description: 'Stop using `go func`, start using `gofuncy`',
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
			{ text: 'About', link: '/' },
			{ text: 'API Reference', link: '/api/' },
		],
		sidebar: [
			{ text: 'About', link: '/' },
			{ text: 'Telemetry', link: '/telemetry' },
			{
				text: 'API Reference',
				items: [
					{ text: 'Go', link: '/go' },
					{ text: 'GoBackground', link: '/gobackground' },
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
				content: 'Stop using `go func`, start using `gofuncy`',
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
