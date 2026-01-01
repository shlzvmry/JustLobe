<script lang="ts">
	import { Send, Menu, MessageSquare, User, Bot, Trash2, Plus } from 'lucide-svelte';
	import { tick, onMount } from 'svelte';
	import SvelteMarkdown from 'svelte-markdown';

	let isSidebarOpen = $state(false);
	let inputText = $state(''); 
	let chatHistory = $state([{ role: 'assistant', content: '你好！我是 LobeChat 助手。' }]);
	let scrollContainer: HTMLElement | undefined = $state();

	onMount(async () => {
		const res = await fetch('http://localhost:8080/api/history');
		if (res.ok) {
			const history = await res.json();
			if (history?.length > 0) chatHistory = history;
			scrollToBottom();
		}
	});

	async function sendMessage() {
		if (!inputText.trim()) return;
		const userMsg = inputText;
		chatHistory.push({ role: 'user', content: userMsg });
		inputText = '';
		await scrollToBottom();

		try {
			chatHistory.push({ role: 'assistant', content: '' });
			const aiIdx = chatHistory.length - 1;

			const res = await fetch('http://localhost:8080/api/chat', {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ message: userMsg })
			});

			const reader = res.body?.getReader();
			const decoder = new TextDecoder();

			if (reader) {
				while (true) {
					const { done, value } = await reader.read();
					if (done) break;
					// 直接追加解码后的字符，保留所有换行和空格
					chatHistory[aiIdx].content += decoder.decode(value, { stream: true });
					scrollToBottom();
				}
			}
		} catch (e) {
			chatHistory.push({ role: 'assistant', content: '连接失败' });
		}
	}

	function handleKeyDown(e: KeyboardEvent) {
		if (e.key === 'Enter' && !e.shiftKey) { e.preventDefault(); sendMessage(); }
	}

	async function scrollToBottom() {
		await tick();
		if (scrollContainer) scrollContainer.scrollTop = scrollContainer.scrollHeight;
	}

	async function clearChat() {
		if (confirm('清空记录？')) {
			await fetch('http://localhost:8080/api/history', { method: 'DELETE' });
			chatHistory = [{ role: 'assistant', content: '已清空' }];
		}
	}
</script>

<div class="flex h-screen w-full bg-[#f8f9fa] text-[#333] overflow-hidden">
	<aside class="fixed inset-y-0 left-0 z-50 w-72 bg-white border-r border-gray-200 transform transition-transform {isSidebarOpen ? 'translate-x-0' : '-translate-x-full'} md:relative md:translate-x-0 md:flex md:flex-col">
		<div class="p-4 border-b flex justify-between items-center">
			<div class="flex items-center gap-2 font-bold text-blue-600">LobeChat Mini</div>
			<button class="md:hidden" onclick={() => isSidebarOpen = false}><Menu size={20} /></button>
		</div>
		<div class="p-3">
			<button onclick={() => chatHistory = [{role:'assistant', content:'新对话'}]} class="w-full bg-blue-600 text-white py-2 rounded-xl flex items-center justify-center gap-2"><Plus size={18}/>新建对话</button>
		</div>
		<nav class="flex-1 p-2"><div class="bg-blue-50 p-3 rounded-xl text-blue-700 flex items-center gap-2"><MessageSquare size={18}/>当前对话</div></nav>
	</aside>

	<main class="flex flex-1 flex-col bg-white relative">
		<header class="h-14 flex items-center justify-between px-4 border-b bg-white/80 backdrop-blur-md sticky top-0 z-30">
			<button class="md:hidden" onclick={() => isSidebarOpen = true}><Menu size={20} /></button>
			<span class="font-bold">DeepSeek-V3</span>
			<button onclick={clearChat} class="text-gray-400 hover:text-red-500"><Trash2 size={18}/></button>
		</header>

		<section bind:this={scrollContainer} class="flex-1 overflow-y-auto p-4 md:p-6 space-y-6">
			<div class="max-w-3xl mx-auto space-y-6">
				{#each chatHistory as msg}
					<div class="flex {msg.role === 'user' ? 'justify-end' : 'justify-start'}">
						<div class="flex max-w-[90%] gap-3 {msg.role === 'user' ? 'flex-row-reverse' : 'flex-row'}">
							<div class="w-8 h-8 rounded-xl flex-shrink-0 flex items-center justify-center {msg.role === 'user' ? 'bg-blue-600 text-white' : 'bg-gray-100'}">
								{#if msg.role === 'user'}<User size={16}/>{:else}<Bot size={16}/>{/if}
							</div>
							<div class="px-4 py-2 rounded-2xl text-sm shadow-sm {msg.role === 'user' ? 'bg-blue-600 text-white' : 'bg-[#f2f3f5] text-gray-800'}">
								<div class="markdown-body">
									<SvelteMarkdown source={msg.content} />
								</div>
							</div>
						</div>
					</div>
				{/each}
			</div>
		</section>

		<footer class="p-4 border-t">
			<div class="max-w-3xl mx-auto flex items-end gap-2 bg-[#f4f4f4] rounded-2xl p-2 focus-within:bg-white border focus-within:border-blue-400 transition-all">
				<textarea bind:value={inputText} onkeydown={handleKeyDown} placeholder="输入消息..." class="flex-1 max-h-32 min-h-[40px] p-2 bg-transparent border-none outline-none text-sm resize-none"></textarea>
				<button onclick={sendMessage} class="p-2 bg-blue-600 text-white rounded-xl disabled:opacity-30" disabled={!inputText.trim()}><Send size={18}/></button>
			</div>
		</footer>
	</main>
</div>

<style>
	/* 核心修复：确保 Markdown 换行和代码块显示正常 */
	.markdown-body :global(pre) { 
		background: #1e1e1e !important; 
		color: #d4d4d4 !important; 
		padding: 1rem; 
		border-radius: 8px; 
		margin: 0.8rem 0;
		overflow-x: auto;
		white-space: pre-wrap; /* 允许长代码换行 */
		word-break: break-all;
	}
	.markdown-body :global(code) { 
		font-family: 'Fira Code', monospace;
		background: rgba(0,0,0,0.05);
		padding: 0.1rem 0.3rem;
		border-radius: 4px;
	}
	.markdown-body :global(pre code) { background: transparent; padding: 0; }
	.markdown-body :global(p) { margin-bottom: 0.6rem; line-height: 1.6; }
	.markdown-body :global(h1), .markdown-body :global(h2), .markdown-body :global(h3) { 
		font-weight: bold; margin: 1rem 0 0.5rem 0; display: block; 
	}
</style>