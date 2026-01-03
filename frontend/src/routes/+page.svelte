<script lang="ts">
	import { Send, Menu, MessageSquare, User, Bot, Trash2, Plus, Square, Crown, Diamond, X, Edit2, Check, Sparkles, History } from 'lucide-svelte';
	import { tick, onMount } from 'svelte';
	import SvelteMarkdown from 'svelte-markdown';

	// --- 核心数据结构定义 ---
	type ChatSession = {
		title: string;
		history: { role: string; content: string }[];
	};

	let isSidebarOpen = $state(false);
	let inputText = $state(''); 
	let scrollContainer: HTMLElement | undefined = $state();
	
	// 当前显示的对话内容
	let chatHistory = $state([{ role: 'assistant', content: '> **SYSTEM NOTICE**\n> 正在绕过付费验证...\n> 开发者模式已激活。\n> \n> 哪怕只有两周，我们也要优雅地完成作业。请下达指令。' }]);
	// 当前显示的标题
	let currentTitle = $state('Project Specification'); 
	
	// --- 历史档案柜 (存储所有后台的对话) ---
	let archivedSessions = $state<ChatSession[]>([]); 
	
	let showRenameModal = $state(false);
	let tempTitle = $state(''); 
	let isRenamingHistoryIndex = $state(-1);
	let isLoading = $state(false);
	let abortController: AbortController | null = null;

	onMount(async () => {
		if (window.innerWidth > 768) isSidebarOpen = true;

		// --- 核心修改：优先从本地读取完整状态 ---
		const savedCurrent = localStorage.getItem('justlobe_current_session');
		const savedArchives = localStorage.getItem('justlobe_archived_sessions');

		if (savedArchives) {
			try { archivedSessions = JSON.parse(savedArchives); } catch(e) {}
		}

		if (savedCurrent) {
			try {
				const session = JSON.parse(savedCurrent);
				currentTitle = session.title;
				chatHistory = session.history;
				scrollToBottom();
			} catch(e) {}
		} else {
			// 兼容旧逻辑
			try {
				const res = await fetch('http://localhost:8080/api/history');
				if (res.ok) {
					const history = await res.json();
					if (history?.length > 0) chatHistory = history;
				}
			} catch(e) {}
		}
	});

	// --- 自动保存功能 ---
	$effect(() => {
		const currentSession = { title: currentTitle, history: chatHistory };
		localStorage.setItem('justlobe_current_session', JSON.stringify(currentSession));
	});

	// --- 新建会话逻辑 ---
	function startNewSession() {
		// 1. 把当前正在聊的，打包扔进档案柜
		if (chatHistory.length > 1 || currentTitle !== 'Untitled Session') {
			const sessionToArchive: ChatSession = {
				title: currentTitle,
				history: [...chatHistory]
			};
			archivedSessions = [sessionToArchive, ...archivedSessions];
			localStorage.setItem('justlobe_archived_sessions', JSON.stringify(archivedSessions));
		}

		// 2. 清空当前屏幕，开始新的
		chatHistory = [{ role: 'assistant', content: '> **NEW SESSION**\n> System ready.' }];
		currentTitle = 'Untitled Session';
		
		// 3. 手机端自动收起
		if (window.innerWidth < 768) isSidebarOpen = false;
	}

	// --- 切换会话逻辑 ---
	function switchSession(index: number) {
		// 1. 归档当前会话
		const targetSession = archivedSessions[index]; 
		const sessionToSave = { title: currentTitle, history: chatHistory };
		const newArchives = [...archivedSessions];
		
		// 移除要切换的目标，把它拿出来
		newArchives.splice(index, 1);
		
		// 把当前的存进去 (如果有效)
		if (sessionToSave.history.length > 1 || sessionToSave.title !== 'Untitled Session') {
			newArchives.unshift(sessionToSave); 
		}
		
		// 更新状态
		archivedSessions = newArchives;
		currentTitle = targetSession.title;
		chatHistory = targetSession.history;
		
		localStorage.setItem('justlobe_archived_sessions', JSON.stringify(archivedSessions));
		if (window.innerWidth < 768) isSidebarOpen = false;
		scrollToBottom();
	}

	function openRename(index = -1) {
		isRenamingHistoryIndex = index;
		tempTitle = index === -1 ? currentTitle : archivedSessions[index].title;
		showRenameModal = true;
	}

	// 处理点击事件，防止冒泡
	function handleRenameClick(e: MouseEvent, index: number) {
		e.stopPropagation(); 
		openRename(index);   
	}

	function confirmRename() {
		if (tempTitle.trim()) {
			if (isRenamingHistoryIndex === -1) {
				currentTitle = tempTitle;
			} else {
				archivedSessions[isRenamingHistoryIndex].title = tempTitle;
				localStorage.setItem('justlobe_archived_sessions', JSON.stringify(archivedSessions));
			}
		}
		showRenameModal = false;
	}

	// --- 发送逻辑 ---
	async function sendMessage() {
		if (!inputText.trim()) return;
		const userMsg = inputText;
		chatHistory.push({ role: 'user', content: userMsg });
		inputText = ''; 
		await scrollToBottom();

		try {
			isLoading = true;
			abortController = new AbortController();

			chatHistory.push({ role: 'assistant', content: '' });
			const aiIdx = chatHistory.length - 1;

			const res = await fetch('http://localhost:8080/api/chat', {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ message: userMsg }),
				signal: abortController.signal
			});

			const reader = res.body?.getReader();
			const decoder = new TextDecoder();

			if (reader) {
				while (true) {
					const { done, value } = await reader.read();
					if (done) break;
					let isNearBottom = false;
					if (scrollContainer) {
						const threshold = 100;
						const position = scrollContainer.scrollTop + scrollContainer.clientHeight;
						const height = scrollContainer.scrollHeight;
						isNearBottom = height - position < threshold;
					}
					chatHistory[aiIdx].content += decoder.decode(value, { stream: true });
					if (isNearBottom) scrollToBottom();
				}
			}
		} catch (e: any) {
			if (e.name === 'AbortError') console.log('用户停止生成');
			else chatHistory.push({ role: 'assistant', content: '> *连接已断开，请检查网络或余额。*' });
		} finally {
			isLoading = false;
			abortController = null;
		}
	}

	function stopGeneration() {
		if (abortController) {
			abortController.abort();
			abortController = null;
			isLoading = false;
			chatHistory[chatHistory.length - 1].content += "\n\n> ⛔ **[SYSTEM]** 用户强制终止了数据流。";
			scrollToBottom();
		}
	}

	function handleKeyDown(e: KeyboardEvent) {
		if (e.key === 'Enter' && !e.shiftKey && !isLoading) { e.preventDefault(); sendMessage(); }
	}

	async function scrollToBottom() {
		await tick();
		if (scrollContainer) scrollContainer.scrollTop = scrollContainer.scrollHeight;
	}

	async function clearChat() {
		if (confirm('警告：此操作将永久抹除当前对话记录。确定执行？')) {
			chatHistory = [{ role: 'assistant', content: '> **SYSTEM**\n> 内存已格式化。请重新开始。' }];
		}
	}
</script>

<svelte:head>
	<link rel="preconnect" href="https://fonts.googleapis.com">
	<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin="anonymous">
	<link href="https://fonts.googleapis.com/css2?family=Lora:ital,wght@0,400;0,500;0,600;1,400&family=Playfair+Display:ital,wght@0,400;0,600;0,700;1,400&display=swap" rel="stylesheet">
</svelte:head>

<div class="h-screen w-full bg-[#141414] text-[#e5e5e5] flex md:grid md:grid-cols-[auto_1fr] overflow-hidden selection:bg-[#DBBA87] selection:text-black relative" style="font-family: 'Lora', serif;">
	
	<!-- 重命名弹窗 -->
	{#if showRenameModal}
		<div class="fixed inset-0 z-[100] flex items-center justify-center bg-black/30 backdrop-blur-[5px] p-4 transition-all">
			<div class="w-full max-w-md bg-[#1a1a1a]/90 border border-[#DBBA87]/50 shadow-[0_0_50px_rgba(219,186,135,0.15)] rounded-2xl p-6 transform transition-all scale-100 backdrop-blur-xl">
				<div class="flex justify-between items-center mb-4">
					<span class="text-[#DBBA87] text-xs font-bold tracking-[0.2em] uppercase font-serif">Modify Alias</span>
					<button onclick={() => showRenameModal = false}><X size={18} class="text-[#666] hover:text-white transition-colors"/></button>
				</div>
				<input 
					bind:value={tempTitle}
					class="w-full bg-[#000]/50 border border-[#333] text-[#e5e5e5] text-lg p-4 rounded-xl focus:border-[#DBBA87] focus:outline-none font-serif mb-6 placeholder-[#444]"
					placeholder="Enter new alias..."
					onkeydown={(e) => e.key === 'Enter' && confirmRename()}
				/>
				<div class="flex justify-end gap-3">
					<button onclick={() => showRenameModal = false} class="px-4 py-2 text-xs text-[#888] hover:text-white transition-colors tracking-widest">CANCEL</button>
					<button onclick={confirmRename} class="px-6 py-2 bg-[#DBBA87] text-black font-bold text-xs rounded-xl hover:bg-[#c9aa7a] transition-colors shadow-lg flex items-center gap-2 tracking-widest">
						<Check size={14} /> CONFIRM
					</button>
				</div>
			</div>
		</div>
	{/if}

	<aside class="bg-[#1a1a1a]/95 backdrop-blur-xl border-r border-[#333] transition-all duration-300 ease-in-out shadow-2xl z-50 overflow-hidden
		fixed inset-y-0 left-0 h-full md:relative md:h-auto
		{isSidebarOpen ? 'translate-x-0 w-64' : '-translate-x-full w-64 md:w-0 md:translate-x-0'}"
		style="background-image: url('data:image/svg+xml;base64,PHN2ZyB3aWR0aD0iODAiIGhlaWdodD0iODAiIHZpZXdCb3g9IjAgMCA4MCA4MCIgeG1sbnM9Imh0dHA6Ly93d3cudzMub3JnLzIwMDAvc3ZnIj48cGF0aCBkPSJNMCAwIEw4MCA4MCBNODAgMCBMMCA4MCIgc3Ryb2tlPSIjRTVFNEUyIiBzdHJva2Utd2lkdGg9IjAuNSIgc3Ryb2tlLWRhc2hhcnJheT0iMiA4IiBzdHJva2Utb3BhY2l0eT0iMC4xIi8+PHBhdGggZD0iTTQwIDM4IEw0MiA0MCBMNDAgNDIgTDM4IDQwIFoiIGZpbGw9IiNiODg2MGIiIGZpbGwtb3BhY2l0eT0iMC4xIi8+PC9zdmc+'); background-size: 80px 80px;">
		
		<div class="h-full w-64 flex flex-col overflow-y-auto relative z-10">
			<div class="p-6 border-b border-[#333] flex justify-between items-start">
				<div class="flex flex-col">
					<span class="font-serif text-2xl font-bold text-[#f5f5f5] tracking-wider italic">JustLobe</span>
					<span class="text-[10px] text-[#DBBA87] uppercase tracking-[0.2em] mt-1">Premium Edition</span>
				</div>
				<button class="md:hidden text-[#666] hover:text-white transition-colors p-1" onclick={() => isSidebarOpen = false} aria-label="Close Sidebar"><X size={24} /></button>
			</div>
			
			<div class="p-3 space-y-4">
				<div class="bg-gradient-to-br from-[#222] to-[#111] border border-[#333] p-3 rounded-xl relative overflow-hidden group shadow-lg mx-1">
					<div class="absolute top-0 right-0 p-2 opacity-20 group-hover:opacity-50 transition-opacity"><Crown size={32} /></div>
					<div class="text-[10px] text-[#888] mb-1 font-bold tracking-wider">CURRENT PLAN</div>
					<div class="text-[#DBBA87] font-bold flex items-center gap-2 text-sm"><Diamond size={14} fill="currentColor"/> Ultimate Pro</div>
					<div class="mt-3 text-[10px] text-[#666] border-t border-[#333] pt-2 flex justify-between items-center whitespace-nowrap">
						<span>EXP: 1 WEEK</span>
						<span class="text-red-400/80 animate-pulse bg-red-900/10 px-1 rounded">URGENT</span>
					</div>
				</div>

				<button onclick={startNewSession} class="w-full border border-[#333] hover:border-[#DBBA87] text-[#888] hover:text-[#DBBA87] py-3 rounded-xl flex items-center justify-center gap-2 transition-all duration-300 text-sm uppercase tracking-widest bg-[#222]/50 hover:bg-[#2a2a2a]">
					<Plus size={16}/> New Thread
				</button>
			</div>
			
			<nav class="flex-1 p-2 space-y-2">
				<div class="text-[10px] text-[#666] px-4 py-2 uppercase tracking-widest">History Logs</div>
				
				<!-- 1. 当前会话 (Active) -->
				<div class="group mx-2 p-3 rounded-xl bg-[#2a2a2a] border border-[#DBBA87]/40 text-[#fff] text-xs flex items-center gap-3 cursor-default relative shadow-lg">
					<MessageSquare size={14} class="text-[#DBBA87]"/> 
					<span class="truncate flex-1 font-serif pr-6 font-bold">{currentTitle}</span>
					<div class="w-1.5 h-1.5 rounded-full bg-green-500 shadow-[0_0_5px_rgba(34,197,94,0.5)]"></div>
				</div>

				<!-- 2. 历史会话列表 (History) -->
				{#each archivedSessions as session, i}
					<div 
						onclick={() => switchSession(i)}
						onkeydown={(e) => e.key === 'Enter' && switchSession(i)}
						role="button"
						tabindex="0"
						class="w-full group mx-2 p-3 rounded-xl bg-transparent border border-transparent text-[#999] text-xs flex items-center gap-3 hover:bg-[#222] hover:border-[#333] cursor-pointer transition-all relative text-left"
					>
						<History size={14} class="text-[#666] group-hover:text-[#DBBA87] transition-colors"/> 
						<span class="truncate flex-1 font-serif pr-6 group-hover:text-[#ccc] transition-colors">{session.title}</span>
						
						<!-- 内部小按钮 -->
						<button 
							type="button"
							onclick={(e) => handleRenameClick(e, i)}
							class="absolute right-2 opacity-0 group-hover:opacity-100 transition-opacity text-[#666] hover:text-[#DBBA87] p-1 bg-transparent border-none cursor-pointer"
							aria-label="Rename"
						>
							<Edit2 size={12} />
						</button>
					</div>
				{/each}
			</nav>

			<div class="p-4 text-[10px] text-[#555] border-t border-[#333] text-center font-serif italic whitespace-nowrap truncate opacity-70 hover:opacity-100 transition-opacity">
				"Powered by Code & Caffeine."
			</div>
		</div>
	</aside>

	{#if isSidebarOpen}
		<button class="fixed inset-0 bg-black/80 z-40 md:hidden backdrop-blur-sm w-full h-full border-none cursor-default" onclick={() => isSidebarOpen = false} aria-label="Close Sidebar"></button>
	{/if}

	<main class="flex flex-1 flex-col relative w-full bg-[#141414] overflow-hidden">
		<!-- 背景纹理 (完全保留原样) -->
		<div class="absolute inset-0 z-0 pointer-events-none" style="background-image: url('data:image/svg+xml;base64,PHN2ZyB3aWR0aD0iNDAiIGhlaWdodD0iNDAiIHZpZXdCb3g9IjAgMCA0MCA0MCIgeG1sbnM9Imh0dHA6Ly93d3cudzMub3JnLzIwMDAvc3ZnIj48IS0tIEdvbGQgTGluZXMgLS0+PHBhdGggZD0iTTAgMCBMNDAgNDAgTTQwIDAgTDAgNDAiIHN0cm9rZT0iI2I4ODYwYiIgc3Ryb2tlLXdpZHRoPSIwLjgiIHN0cm9rZS1kYXNoYXJyYXk9IjQgMTYiIHN0cm9rZS1saW5lY2FwPSJyb3VuZCIgc3Ryb2tlLW9wYWNpdHk9IjAuMjUiLz48IS0tIFNpbHZlciBMaW5lcyAoT2Zmc2V0KSAtLT48cGF0aCBkPSJNMCAwIEw0MCA0MCBNNDAgMCBMMCA0MCIgc3Ryb2tlPSIjRTVFNEUyIiBzdHJva2Utd2lkdGg9IjAuOCIgc3Ryb2tlLWRhc2hhcnJheT0iNCAxNiIgc3Ryb2tlLWRhc2hvZmZzZXQ9IjEwIiBzdHJva2UtbGluZWNhcD0icm91bmQiIHN0cm9rZS1vcGFjaXR5PSIwLjE1Ii8+PCEtLSBEb3RzIC0tPjxjaXJjbGUgY3g9IjIwIiBjeT0iMjAiIHI9IjEiIGZpbGw9IiNiODg2MGIiIGZpbGwtb3BhY2l0eT0iMC4zIi8+PGNpcmNsZSBjeD0iMCIgY3k9IjIwIiByPSIwLjgiIGZpbGw9IiNFNUU0RTIiIGZpbGwtb3BhY2l0eT0iMC4yIi8+PGNpcmNsZSBjeD0iMjAiIGN5PSIwIiByPSIwLjgiIGZpbGw9IiNFNUU0RTIiIGZpbGwtb3BhY2l0eT0iMC4yIi8+PC9zdmc+'); background-size: 40px 40px; opacity: 0.4;"></div>
		
		<header class="h-16 flex items-center justify-between px-6 border-b border-[#222] bg-[#141414]/80 backdrop-blur-md sticky top-0 z-30">
			<div class="flex items-center gap-4">
				<button class="p-2 border border-[#333] rounded-xl bg-[#1a1a1a] text-[#888] hover:bg-[#DBBA87] hover:text-black hover:border-[#DBBA87] transition-all shadow-sm" onclick={() => isSidebarOpen = !isSidebarOpen} aria-label="Toggle Sidebar"><Menu size={20} /></button>
				<button class="flex flex-col group cursor-pointer text-left bg-transparent border-none p-0" onclick={() => openRename(-1)} aria-label="Rename conversation">
					<div class="flex items-center gap-2">
						<span class="font-serif font-bold text-[#e5e5e5] text-lg leading-none group-hover:text-[#DBBA87] transition-colors">{currentTitle}</span>
						<Edit2 size={12} class="opacity-0 group-hover:opacity-50 text-[#888] transition-opacity"/>
					</div>
					<div class="flex items-center gap-1.5 mt-0.5">
						<div class="w-1.5 h-1.5 rounded-full bg-green-500 shadow-[0_0_5px_rgba(34,197,94,0.5)]"></div>
						<span class="font-serif italic text-[10px] font-bold tracking-widest text-[#666]">DEEPSEEK V3</span>
					</div>
				</button>
			</div>
			<button onclick={clearChat} class="text-[#666] hover:text-red-400 transition-colors"><Trash2 size={18}/></button>
		</header>

		<section bind:this={scrollContainer} class="flex-1 overflow-y-auto p-4 md:p-8 space-y-8 scroll-smooth">
			<div class="max-w-4xl mx-auto space-y-8">
				{#each chatHistory as msg}
					<div class="flex {msg.role === 'user' ? 'justify-end' : 'justify-start'} group relative">
						<div class="flex max-w-[90%] md:max-w-[80%] gap-3 md:gap-4 {msg.role === 'user' ? 'flex-row-reverse' : 'flex-row'}">
							<div class="w-8 h-8 md:w-10 md:h-10 flex-shrink-0 flex items-center justify-center border border-[#333] rounded-full {msg.role === 'user' ? 'bg-[#222]' : 'bg-[#000]'} shadow-md">
								{#if msg.role === 'user'}<User size={16} class="text-[#ccc]"/>{:else}<Bot size={16} class="text-[#DBBA87]"/>{/if}
							</div>
							<div class="px-4 py-3 md:px-5 md:py-3 rounded-2xl border shadow-sm flex items-center overflow-hidden min-w-0 w-fit {msg.role === 'user' ? 'bg-[#222] border-[#333] text-[#e5e5e5]' : 'bg-[#1a1a1a] border-[#333]/50 text-[#d4d4d4]'}">
								<div class="markdown-body font-light w-full break-words"><SvelteMarkdown source={msg.content} /></div>
							</div>
							{#if msg.role === 'assistant'}
								<div class="hidden group-hover:block absolute -bottom-5 left-16 text-[10px] text-[#666] font-serif italic">TOKEN_ID: {Math.random().toString(36).substr(2, 9).toUpperCase()}</div>
							{/if}
						</div>
					</div>
				{/each}
			</div>
		</section>

		<footer class="p-6">
			<div class="max-w-4xl mx-auto">
				<div class="relative group bg-[#262626] border border-[#444] rounded-3xl focus-within:border-[#DBBA87] focus-within:shadow-[0_0_20px_rgba(219,186,135,0.2)] transition-all duration-300 overflow-hidden">
					<div class="flex items-end p-2">
						<textarea bind:value={inputText} onkeydown={handleKeyDown} placeholder="Input command sequence..." class="flex-1 max-h-40 min-h-[50px] p-4 bg-transparent border-none outline-none text-base resize-none placeholder-[#a3a3a3] text-white" disabled={isLoading}></textarea>
						<div class="p-2">
							{#if isLoading}
								<button onclick={stopGeneration} class="p-3 bg-red-900/40 text-red-200 hover:bg-red-600 hover:text-white border border-red-500/50 rounded-full transition-all"><Square size={20} fill="currentColor" /></button>
							{:else}
								<button onclick={sendMessage} class="p-3 bg-[#444] text-white hover:bg-[#DBBA87] hover:text-black rounded-full transition-all disabled:opacity-30 disabled:cursor-not-allowed shadow-lg" disabled={!inputText.trim()}><Send size={20}/></button>
							{/if}
						</div>
					</div>
				</div>
				<div class="text-center mt-3 text-[10px] text-[#666] font-serif transition-all duration-300">
					<span class="md:hidden tracking-wider">JUSTLOBE © 2026</span>
					<span class="hidden md:inline tracking-widest">JUSTLOBE SYSTEM © 2026. ALL RIGHTS RESERVED.</span>
				</div>
			</div>
		</footer>
	</main>
</div>

<style>
	:global(::-webkit-scrollbar) { width: 6px; }
	:global(::-webkit-scrollbar-track) { background: #141414; }
	:global(::-webkit-scrollbar-thumb) { background: #333; border-radius: 3px; }
	:global(::-webkit-scrollbar-thumb:hover) { background: #DBBA87; }
	.markdown-body :global(pre) { background: #000 !important; border: none !important; color: #DBBA87 !important; padding: 1rem; margin: 0.8rem 0; overflow-x: auto; font-family: 'Lora', serif !important; font-size: 0.95em; border-radius: 12px; }
	.markdown-body :global(code) { font-family: 'Lora', serif !important; background: transparent !important; border: none !important; padding: 0; color: inherit; }
	.markdown-body :global(p code), .markdown-body :global(li code) { background: rgba(255,255,255,0.1) !important; padding: 0.1rem 0.4rem !important; border-radius: 4px; color: #fff !important; }
	.markdown-body :global(p) { margin-bottom: 0; line-height: 1.7; font-family: 'Lora', serif; }
	.markdown-body :global(h1), .markdown-body :global(h2), .markdown-body :global(h3) { font-family: 'Playfair Display', serif; font-weight: 600; color: #fff; margin: 1.5rem 0 0.5rem 0; }
	.markdown-body :global(blockquote) { border-left: 3px solid #DBBA87; padding-left: 1rem; margin: 0.5rem 0; color: #999; font-style: italic; }
	.markdown-body :global(strong) { color: #fff; font-weight: 600; }
	.markdown-body :global(ul), .markdown-body :global(ol) { padding-left: 1.2rem; margin: 0.5rem 0; }
</style>