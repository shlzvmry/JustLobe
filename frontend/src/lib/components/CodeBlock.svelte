<script lang="ts">
	import { Copy, Check } from 'lucide-svelte';
	import hljs from 'highlight.js';
	import { tick } from 'svelte';

	// svelte-markdown 传进来的参数名是 text (内容) 和 lang (语言)
	let { text, lang } = $props(); 
	let copied = $state(false);
	let codeElement: HTMLElement | undefined = $state();

	async function copyCode() {
		await navigator.clipboard.writeText(text);
		copied = true;
		setTimeout(() => (copied = false), 2000);
	}

	// 高亮逻辑：在流式生成时，确保文字始终可见
	$effect(() => {
		if (text && codeElement) {
			// 1. 强制文字颜色，防止隐身
			codeElement.style.color = '#a9b1d6'; 
			
			// 2. 异步触发高亮
			tick().then(() => {
				if (codeElement) {
					codeElement.removeAttribute('data-highlighted');
					hljs.highlightElement(codeElement);
				}
			});
		}
	});
</script>

<div class="relative my-6 rounded-xl border border-[#333] bg-[#1a1b26] overflow-hidden shadow-2xl">
	<!-- 顶部工具栏 -->
	<div class="flex items-center justify-between px-4 py-1.5 bg-[#24283b] border-b border-[#333] select-none">
		<span class="text-[10px] font-bold uppercase tracking-widest text-[#7aa2f7] font-mono">
			{lang || 'code'}
		</span>
		<button onclick={copyCode} class="flex items-center gap-1.5 text-[#666] hover:text-[#DBBA87] transition-colors p-1">
			{#if copied}
				<span class="text-[10px] font-bold text-green-500">COPIED!</span>
				<Check size={14} class="text-green-500" />
			{:else}
				<Copy size={14} />
			{/if}
		</button>
	</div>

	<!-- 代码内容：直接渲染 {text} -->
	<pre class="p-4 overflow-x-auto !bg-transparent !m-0"><code 
		bind:this={codeElement} 
		class="language-{lang} !text-[#a9b1d6] font-mono text-sm leading-relaxed block whitespace-pre"
		style="color: #a9b1d6 !important;"
	>{text}</code></pre>
</div>

<style>
	pre::-webkit-scrollbar { height: 4px; }
	pre::-webkit-scrollbar-thumb { background: #333; border-radius: 10px; }
	pre::-webkit-scrollbar-thumb:hover { background: #DBBA87; }
</style>