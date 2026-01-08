<script lang="ts">
	import { User, Bot, Copy, FileText, RefreshCw, Pencil, Check, X, FileType, ChevronLeft, ChevronRight, Play, Square } from 'lucide-svelte';
	import SvelteMarkdown from 'svelte-markdown';
	import { slide } from 'svelte/transition';
	import { tick } from 'svelte'; 
	import hljs from 'highlight.js';
	import 'highlight.js/styles/tokyo-night-dark.css'; 
	import CodeBlock from './CodeBlock.svelte';

	let { msg, index, isLast, onEdit, onRegenerate, onSwitchVersion, onContinue } = $props();

	let isEditing = $state(false);
	let copyStatus = $state('idle'); 
	let chatContainer: HTMLElement | undefined = $state(); 

	let currentContent = $derived(msg.versions ? msg.versions[msg.currentVersion || 0] : msg.content);
	let versionCount = $derived(msg.versions ? msg.versions.length : 1);
	let currentVerIdx = $derived(msg.currentVersion || 0);
	let editContent = $state('');
	
	// 定义渲染器
	const renderers: any = {
		code: CodeBlock
	};

	// 同步编辑内容
	$effect(() => {
		if (!isEditing) {
			editContent = currentContent;
		}
	});

	function copyText() {
		navigator.clipboard.writeText(currentContent);
		copyStatus = 'copied';
		setTimeout(() => copyStatus = 'idle', 2000);
	}

	function exportTxt() {
		const blob = new Blob([currentContent], { type: 'text/plain' });
		const url = URL.createObjectURL(blob);
		const a = document.createElement('a');
		a.href = url;
		a.download = `JustLobe_Export_${Date.now()}.txt`;
		a.click();
	}

	function exportDoc() {
		const htmlContent = `<html><head><meta charset='utf-8'></head><body>${currentContent.replace(/\n/g, '<br>')}</body></html>`;
		const blob = new Blob([htmlContent], { type: 'application/msword' });
		const url = URL.createObjectURL(blob);
		const a = document.createElement('a');
		a.href = url;
		a.download = `JustLobe_Export_${Date.now()}.doc`;
		a.click();
	}

	function submitEdit() {
		isEditing = false;
		onEdit(index, editContent);
	}
</script>

<div class="flex flex-col mb-6 group relative">
	<div class="flex {msg.role === 'user' ? 'justify-end' : 'justify-start'}">
		<div class="flex max-w-[95%] md:max-w-[85%] gap-3 md:gap-4 {msg.role === 'user' ? 'flex-row-reverse' : 'flex-row'}">
			
			<div class="w-8 h-8 md:w-10 md:h-10 flex-shrink-0 flex items-center justify-center border border-[#333] rounded-full {msg.role === 'user' ? 'bg-[#222]' : 'bg-[#000]'} shadow-md mt-1">
				{#if msg.role === 'user'}
					<User size={16} class="text-[#ccc]"/>
				{:else}
					<Bot size={16} class="text-[#DBBA87]"/>
				{/if}
			</div>

			<div class="flex flex-col gap-1 min-w-0 max-w-full">
				<div class="px-4 py-3 md:px-5 md:py-4 rounded-2xl border shadow-sm relative overflow-hidden transition-all
					{msg.role === 'user' ? 'bg-[#222] border-[#333] text-[#e5e5e5]' : 'bg-[#1a1a1a] border-[#333]/50 text-[#d4d4d4]'}">
					
					{#if isEditing}
						<div class="flex flex-col gap-2 min-w-[300px]" transition:slide>
							<textarea bind:value={editContent} class="w-full bg-[#111] border border-[#DBBA87] rounded-lg p-2 text-sm text-[#e5e5e5] focus:outline-none min-h-[100px]"></textarea>
							<div class="flex justify-end gap-2">
								<button onclick={() => isEditing = false} class="p-1 text-[#666] hover:text-white"><X size={16}/></button>
								<button onclick={submitEdit} class="p-1 text-[#DBBA87] hover:text-white bg-[#DBBA87]/20 rounded"><Check size={16}/></button>
							</div>
						</div>
					{:else}
						<div bind:this={chatContainer} class="markdown-body font-light w-full break-words text-sm md:text-base leading-relaxed">
							<SvelteMarkdown source={currentContent} renderers={renderers as any} />
						</div>
					{/if}

					{#if versionCount > 1 && !isEditing}
						<div class="flex items-center gap-2 mt-3 pt-2 border-t border-[#333]/50 select-none">
							<button disabled={currentVerIdx === 0} onclick={() => onSwitchVersion(index, currentVerIdx - 1)} class="text-[#666] hover:text-[#DBBA87] disabled:opacity-30 disabled:hover:text-[#666]"><ChevronLeft size={14}/></button>
							<span class="text-[10px] font-mono text-[#666]">{currentVerIdx + 1} / {versionCount}</span>
							<button disabled={currentVerIdx === versionCount - 1} onclick={() => onSwitchVersion(index, currentVerIdx + 1)} class="text-[#666] hover:text-[#DBBA87] disabled:opacity-30 disabled:hover:text-[#666]"><ChevronRight size={14}/></button>
						</div>
					{/if}
				</div>

				{#if msg.isStopped && isLast && msg.role === 'assistant'}
					<div class="flex items-center gap-3 pl-2 mt-1" transition:slide>
						<div class="flex items-center gap-1 text-[#666] italic text-xs"><Square size={10} fill="currentColor"/><span class="font-serif">Stop</span></div>
						<button onclick={() => onContinue(index)} class="flex items-center gap-1 text-[#DBBA87] hover:text-white text-xs transition-colors border border-[#DBBA87]/30 hover:border-[#DBBA87] px-2 py-0.5 rounded-full"><Play size={10} fill="currentColor"/><span class="uppercase tracking-wider font-bold">Continue</span></button>
					</div>
				{/if}

				{#if !isEditing && currentContent}
					<div class="flex gap-1 opacity-0 group-hover:opacity-100 transition-opacity duration-200 {msg.role === 'user' ? 'flex-row-reverse' : 'flex-row'} px-1 mt-1">
						<button onclick={copyText} class="p-1.5 text-[#666] hover:text-[#DBBA87] rounded hover:bg-[#222] transition-colors" title="Copy">{#if copyStatus === 'copied'} <Check size={14}/> {:else} <Copy size={14}/> {/if}</button>
						{#if msg.role === 'assistant'}
							<button onclick={exportTxt} class="p-1.5 text-[#666] hover:text-[#DBBA87] rounded hover:bg-[#222] transition-colors" title="Export TXT"><FileText size={14}/></button>
							<button onclick={exportDoc} class="p-1.5 text-[#666] hover:text-[#DBBA87] rounded hover:bg-[#222] transition-colors" title="Export DOC"><FileType size={14}/></button>
							<button onclick={() => onRegenerate(index)} class="p-1.5 text-[#666] hover:text-[#DBBA87] rounded hover:bg-[#222] transition-colors" title="Regenerate"><RefreshCw size={14}/></button>
						{/if}
						{#if msg.role === 'user'}
							<button onclick={() => isEditing = true} class="p-1.5 text-[#666] hover:text-[#DBBA87] rounded hover:bg-[#222] transition-colors" title="Edit"><Pencil size={14}/></button>
						{/if}
					</div>
				{/if}
			</div>
		</div>
	</div>
</div>

<style>
	.markdown-body :global(p code), .markdown-body :global(li code) { 
		background: rgba(255,255,255,0.1) !important; 
		padding: 0.1rem 0.4rem !important; 
		border-radius: 4px; 
		color: #f7768e !important; 
	}
</style>