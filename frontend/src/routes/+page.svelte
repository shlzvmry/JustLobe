<script lang="ts">
	import { Send, Menu, MessageSquare, Trash2, Plus, Square, Crown, Diamond, X, Edit2, Check, History } from 'lucide-svelte';
	import { tick, onMount } from 'svelte';
	import ChatItem from '$lib/components/ChatItem.svelte';

	// --- 数据结构 ---
	type Message = {
		role: string;
		content: string;
		versions?: string[];
		currentVersion?: number;
		isStopped?: boolean;
	};

	type ChatSession = {
		title: string;
		history: Message[];
	};

	// --- 状态管理 ---
	let isSidebarOpen = $state(false);
	let inputText = $state(''); 
	let scrollContainer: HTMLElement | undefined = $state();
	let sessions = $state<ChatSession[]>([
		{ 
			title: 'Project Specification', 
			history: [{ role: 'assistant', content: '> **SYSTEM NOTICE**\n> 开发者模式已激活。\n> \n> 请下达指令。' }] 
		}
	]);
	let activeIdx = $state(0);
	let chatHistory = $derived(sessions[activeIdx].history);
	let currentTitle = $derived(sessions[activeIdx].title);
	
	let showRenameModal = $state(false);
	let tempTitle = $state(''); 
	let isRenamingHistoryIndex = $state(-1);
	let isLoading = $state(false);
	let abortController: AbortController | null = null;
	let generatingSession = $state<ChatSession | null>(null); // 记录哪个对话正在生成

	onMount(() => {
		if (window.innerWidth > 768) isSidebarOpen = true;
		const saved = localStorage.getItem('justlobe_all_sessions_v2');
		if (saved) {
			try { sessions = JSON.parse(saved); } catch(e) {}
		}
	});

	$effect(() => {
		localStorage.setItem('justlobe_all_sessions_v2', JSON.stringify(sessions));
	});

	// --- 核心逻辑 ---

	function moveCurrentToTop() {
		if (activeIdx === 0) return;
		const [current] = sessions.splice(activeIdx, 1);
		sessions.unshift(current);
		activeIdx = 0;
	}

	function getMsgContent(msg: Message) {
		if (msg.versions && msg.versions.length > 0) {
			return msg.versions[msg.currentVersion || 0];
		}
		return msg.content;
	}

	// 核心修复：显式传入 session 对象
	function setMsgContent(session: ChatSession, msgIdx: number, content: string, createNewVersion = false) {
		const msg = session.history[msgIdx];
		if (!msg) return;
		if (!msg.versions || msg.versions.length === 0) {
			msg.versions = [msg.content || ''];
			msg.currentVersion = 0;
		}
		if (createNewVersion) {
			msg.versions.push(content);
			msg.currentVersion = msg.versions.length - 1;
		} else {
			const verIdx = msg.currentVersion || 0;
			msg.versions[verIdx] = content;
		}
		msg.content = content; 
	}

	async function streamResponse(payloadMsg: string, targetMsgIndex: number, targetSession: ChatSession) {
		try {
			isLoading = true;
			generatingSession = targetSession;
			abortController = new AbortController();
			const res = await fetch('http://localhost:8080/api/chat', {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ message: payloadMsg }), 
				signal: abortController.signal
			});
			const reader = res.body?.getReader();
			const decoder = new TextDecoder();
			if (reader) {
				while (true) {
					const { done, value } = await reader.read();
					if (done) break;
					const chunk = decoder.decode(value, { stream: true });
					
					// 1. 滚动判断：仅当用户正在看这个对话时才计算
					const threshold = 100;
					const isAtBottom = scrollContainer && (targetSession === sessions[activeIdx])
						? (scrollContainer.scrollHeight - scrollContainer.scrollTop - scrollContainer.clientHeight < threshold)
						: false;

					// 2. 更新内容
					const currentText = getMsgContent(targetSession.history[targetMsgIndex]);
					setMsgContent(targetSession, targetMsgIndex, currentText + chunk, false);

					// 3. 执行滚动
					if (isAtBottom && scrollContainer) {
						await tick();
						scrollContainer.scrollTop = scrollContainer.scrollHeight;
					}
				}
			}
		} catch (e: any) {
			if (e.name === 'AbortError') targetSession.history[targetMsgIndex].isStopped = true;
			else setMsgContent(targetSession, targetMsgIndex, getMsgContent(targetSession.history[targetMsgIndex]) + '\n> *Connection interrupted.*', false);
		} finally {
			isLoading = false;
			abortController = null;
			generatingSession = null;
		}
	}

	async function sendMessage() {
		if (!inputText.trim()) return;
		const currentSession = sessions[activeIdx]; 
		moveCurrentToTop();
		const userMsg = inputText;
		currentSession.history.push({ role: 'user', content: userMsg, versions: [userMsg], currentVersion: 0 });
		inputText = ''; 
		await scrollToBottom();
		currentSession.history.push({ role: 'assistant', content: '', versions: [''], currentVersion: 0, isStopped: false });
		await streamResponse(userMsg, currentSession.history.length - 1, currentSession);
	}

	async function handleRegenerate(index: number) {
		if (isLoading) return;
		const currentSession = sessions[activeIdx];
		const userMsg = getMsgContent(currentSession.history[index - 1]);
		setMsgContent(currentSession, index, '', true);
		currentSession.history[index].isStopped = false;
		await streamResponse(userMsg, index, currentSession);
	}

	async function handleEdit(index: number, newContent: string) {
		if (isLoading) return;
		const currentSession = sessions[activeIdx];
		moveCurrentToTop(); 
		setMsgContent(currentSession, index, newContent, true);
		const aiMsgIndex = index + 1;
		if (currentSession.history[aiMsgIndex]?.role === 'assistant') {
			setMsgContent(currentSession, aiMsgIndex, '', true);
			currentSession.history[aiMsgIndex].isStopped = false;
			await streamResponse(newContent, aiMsgIndex, currentSession);
		} else {
			currentSession.history = currentSession.history.slice(0, index + 1);
			currentSession.history.push({ role: 'assistant', content: '', versions: [''], currentVersion: 0, isStopped: false });
			await streamResponse(newContent, currentSession.history.length - 1, currentSession);
		}
	}

	async function handleContinue(index: number) {
		if (isLoading) return;
		const currentSession = sessions[activeIdx];
		currentSession.history[index].isStopped = false;
		const currentText = getMsgContent(currentSession.history[index]);
		await streamResponse(`Please continue: "${currentText}"`, index, currentSession);
	}

	function stopGeneration() {
		if (abortController) {
			abortController.abort();
			isLoading = false;
			if (generatingSession) {
				generatingSession.history[generatingSession.history.length - 1].isStopped = true;
			}
		}
	}

	// --- 其他 UI 函数 ---
	function startNewSession() {
		sessions.unshift({
			title: 'Untitled Session',
			history: [{ role: 'assistant', content: '> **NEW SESSION**\n> System ready.' }]
		});
		activeIdx = 0;
		if (window.innerWidth < 768) isSidebarOpen = false;
	}

	function switchSession(index: number) {
		activeIdx = index;
		if (window.innerWidth < 768) isSidebarOpen = false;
		scrollToBottom();
	}

	function openRename(index = -1) {
		isRenamingHistoryIndex = index;
		tempTitle = index === -1 ? sessions[activeIdx].title : sessions[index].title;
		showRenameModal = true;
	}

	function confirmRename() {
		if (tempTitle.trim()) {
			const targetIdx = isRenamingHistoryIndex === -1 ? activeIdx : isRenamingHistoryIndex;
			sessions[targetIdx].title = tempTitle;
		}
		showRenameModal = false;
	}

	function deleteSession(index: number) {
		if (sessions.length <= 1) {
			alert('至少需要保留一个对话窗口。');
			return;
		}
		
		if (confirm('确定要删除这个对话吗？此操作不可撤销。')) {
			// 如果删除的是当前正在生成的对话，先停止它
			if (generatingSession === sessions[index]) {
				stopGeneration();
			}

			sessions.splice(index, 1);

			// 处理索引重置逻辑
			if (index === activeIdx) {
				// 如果删的是当前选中的，跳到前一个或第一个
				activeIdx = Math.max(0, index - 1);
			} else if (index < activeIdx) {
				// 如果删的是当前选中之前的，索引需要减 1 保持指向正确
				activeIdx--;
			}
		}
	}

	function handleSwitchVersion(index: number, versionIdx: number) {
		const msg = sessions[activeIdx].history[index];
		if (!msg || !msg.versions || versionIdx >= msg.versions.length) return;
		msg.currentVersion = versionIdx;
		msg.content = msg.versions[versionIdx];
	}

	function handleKeyDown(e: KeyboardEvent) {
		if (e.key === 'Enter' && !e.shiftKey && !isLoading) { e.preventDefault(); sendMessage(); }
	}

	async function scrollToBottom() {
		await tick();
		if (scrollContainer) scrollContainer.scrollTop = scrollContainer.scrollHeight;
	}

	async function clearChat() {
		if (confirm('确定清空当前对话？')) {
			sessions[activeIdx].history = [{ role: 'assistant', content: '> **SYSTEM**\n> 内存已格式化。' }];
		}
	}
</script>

<!-- HTML & Style 部分保持不变 -->
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
				<input bind:value={tempTitle} class="w-full bg-[#000]/50 border border-[#333] text-[#e5e5e5] text-lg p-4 rounded-xl focus:border-[#DBBA87] focus:outline-none font-serif mb-6 placeholder-[#444]" placeholder="Enter new alias..." onkeydown={(e) => e.key === 'Enter' && confirmRename()}/>
				<div class="flex justify-end gap-3">
					<button onclick={() => showRenameModal = false} class="px-4 py-2 text-xs text-[#888] hover:text-white transition-colors tracking-widest">CANCEL</button>
					<button onclick={confirmRename} class="px-6 py-2 bg-[#DBBA87] text-black font-bold text-xs rounded-xl hover:bg-[#c9aa7a] transition-colors shadow-lg flex items-center gap-2 tracking-widest"><Check size={14} /> CONFIRM</button>
				</div>
			</div>
		</div>
	{/if}

	<aside class="bg-[#1a1a1a]/95 backdrop-blur-xl border-r border-[#333] transition-all duration-300 ease-in-out shadow-2xl z-50 overflow-hidden fixed inset-y-0 left-0 h-full md:relative md:h-auto {isSidebarOpen ? 'translate-x-0 w-64' : '-translate-x-full w-64 md:w-0 md:translate-x-0'}" style="background-image: url('data:image/svg+xml;base64,PHN2ZyB3aWR0aD0iODAiIGhlaWdodD0iODAiIHZpZXdCb3g9IjAgMCA4MCA4MCIgeG1sbnM9Imh0dHA6Ly93d3cudzMub3JnLzIwMDAvc3ZnIj48cGF0aCBkPSJNMCAwIEw4MCA4MCBNODAgMCBMMCA4MCIgc3Ryb2tlPSIjRTVFNEUyIiBzdHJva2Utd2lkdGg9IjAuNSIgc3Ryb2tlLWRhc2hhcnJheT0iMiA4IiBzdHJva2Utb3BhY2l0eT0iMC4xIi8+PHBhdGggZD0iTTQwIDM4IEw0MiA0MCBMNDAgNDIgTDM4IDQwIFoiIGZpbGw9IiNiODg2MGIiIGZpbGwtb3BhY2l0eT0iMC4xIi8+PC9zdmc+'); background-size: 80px 80px;">
		<div class="h-full w-64 flex flex-col overflow-y-auto relative z-10">
			<div class="p-6 border-b border-[#333] flex justify-between items-start">
				<div class="flex flex-col"><span class="font-serif text-2xl font-bold text-[#f5f5f5] tracking-wider italic">JustLobe</span><span class="text-[10px] text-[#DBBA87] uppercase tracking-[0.2em] mt-1">Premium Edition</span></div>
				<button class="md:hidden text-[#666] hover:text-white transition-colors p-1" onclick={() => isSidebarOpen = false} aria-label="Close Sidebar"><X size={24} /></button>
			</div>
			<div class="p-3 space-y-4">
				<div class="bg-gradient-to-br from-[#222] to-[#111] border border-[#333] p-3 rounded-xl relative overflow-hidden group shadow-lg mx-1">
					<div class="absolute top-0 right-0 p-2 opacity-20 group-hover:opacity-50 transition-opacity"><Crown size={32} /></div>
					<div class="text-[10px] text-[#888] mb-1 font-bold tracking-wider">CURRENT PLAN</div>
					<div class="text-[#DBBA87] font-bold flex items-center gap-2 text-sm"><Diamond size={14} fill="currentColor"/> Ultimate Pro</div>
					<div class="mt-3 text-[10px] text-[#666] border-t border-[#333] pt-2 flex justify-between items-center whitespace-nowrap"><span>EXP: 1 WEEK</span><span class="text-red-400/80 animate-pulse bg-red-900/10 px-1 rounded">URGENT</span></div>
				</div>
				<button onclick={startNewSession} class="w-full border border-[#333] hover:border-[#DBBA87] text-[#888] hover:text-[#DBBA87] py-3 rounded-xl flex items-center justify-center gap-2 transition-all duration-300 text-sm uppercase tracking-widest bg-[#222]/50 hover:bg-[#2a2a2a]"><Plus size={16}/> New Thread</button>
			</div>
			<nav class="flex-1 p-2 space-y-2">
    <div class="text-[10px] text-[#666] px-4 py-2 uppercase tracking-widest">History Logs</div>
    
    {#each sessions as session, i}
        <div 
            onclick={() => switchSession(i)} 
            onkeydown={(e) => e.key === 'Enter' && switchSession(i)} 
            role="button" 
            tabindex="0" 
            class="w-full group mx-2 p-3 rounded-xl border transition-all relative text-left flex items-center gap-3 cursor-pointer
            {i === activeIdx 
                ? 'bg-[#2a2a2a] border-[#DBBA87]/40 text-[#fff] shadow-lg' 
                : 'bg-transparent border-transparent text-[#999] hover:bg-[#222] hover:border-[#333]'}"
        >
            {#if i === activeIdx}
                <MessageSquare size={14} class="text-[#DBBA87]"/>
            {:else}
                <History size={14} class="text-[#666] group-hover:text-[#DBBA87] transition-colors"/>
            {/if}

            <span class="truncate flex-1 font-serif pr-12 {i === activeIdx ? 'font-bold' : 'group-hover:text-[#ccc]'}">
                {session.title}
            </span>

            {#if i === activeIdx}
                <div class="w-1.5 h-1.5 rounded-full bg-green-500 shadow-[0_0_5px_rgba(34,197,94,0.5)]"></div>
            {/if}

            <!-- 操作按钮容器（仅在悬停时显示） -->
            <div class="absolute right-2 flex items-center gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
                <!-- 重命名按钮 -->
                <button 
                    type="button" 
                    onclick={(e) => { e.stopPropagation(); openRename(i); }} 
                    class="text-[#666] hover:text-[#DBBA87] p-1 bg-transparent border-none cursor-pointer"
                    title="Rename"
                >
                    <Edit2 size={12} />
                </button>
                <!-- 删除按钮 -->
                <button 
                    type="button" 
                    onclick={(e) => { e.stopPropagation(); deleteSession(i); }} 
                    class="text-[#666] hover:text-red-400 p-1 bg-transparent border-none cursor-pointer"
                    title="Delete"
                >
                    <Trash2 size={12} />
                </button>
            </div>
        </div>
    {/each}
</nav>
			<div class="p-4 text-[10px] text-[#555] border-t border-[#333] text-center font-serif italic whitespace-nowrap truncate opacity-70 hover:opacity-100 transition-opacity">"Powered by Code & Caffeine."</div>
		</div>
	</aside>

	{#if isSidebarOpen}<button class="fixed inset-0 bg-black/80 z-40 md:hidden backdrop-blur-sm w-full h-full border-none cursor-default" onclick={() => isSidebarOpen = false} aria-label="Close Sidebar"></button>{/if}

	<main class="flex flex-1 flex-col relative w-full bg-[#141414] overflow-hidden">
		<div class="absolute inset-0 z-0 pointer-events-none" style="background-image: url('data:image/svg+xml;base64,PHN2ZyB3aWR0aD0iNDAiIGhlaWdodD0iNDAiIHZpZXdCb3g9IjAgMCA0MCA0MCIgeG1sbnM9Imh0dHA6Ly93d3cudzMub3JnLzIwMDAvc3ZnIj48IS0tIEdvbGQgTGluZXMgLS0+PHBhdGggZD0iTTAgMCBMNDAgNDAgTTQwIDAgTDAgNDAiIHN0cm9rZT0iI2I4ODYwYiIgc3Ryb2tlLXdpZHRoPSIwLjgiIHN0cm9rZS1kYXNoYXJyYXk9IjQgMTYiIHN0cm9rZS1saW5lY2FwPSJyb3VuZCIgc3Ryb2tlLW9wYWNpdHk9IjAuMjUiLz48IS0tIFNpbHZlciBMaW5lcyAoT2Zmc2V0KSAtLT48cGF0aCBkPSJNMCAwIEw0MCA0MCBNNDAgMCBMMCA0MCIgc3Ryb2tlPSIjRTVFNEUyIiBzdHJva2Utd2lkdGg9IjAuOCIgc3Ryb2tlLWRhc2hhcnJheT0iNCAxNiIgc3Ryb2tlLWRhc2hvZmZzZXQ9IjEwIiBzdHJva2UtbGluZWNhcD0icm91bmQiIHN0cm9rZS1vcGFjaXR5PSIwLjE1Ii8+PCEtLSBEb3RzIC0tPjxjaXJjbGUgY3g9IjIwIiBjeT0iMjAiIHI9IjEiIGZpbGw9IiNiODg2MGIiIGZpbGwtb3BhY2l0eT0iMC4zIi8+PGNpcmNsZSBjeD0iMCIgY3k9IjIwIiByPSIwLjgiIGZpbGw9IiNFNUU0RTIiIGZpbGwtb3BhY2l0eT0iMC4yIi8+PGNpcmNsZSBjeD0iMjAiIGN5PSIwIiByPSIwLjgiIGZpbGw9IiNFNUU0RTIiIGZpbGwtb3BhY2l0eT0iMC4yIi8+PC9zdmc+'); background-size: 40px 40px; opacity: 0.4;"></div>
		
		<header class="h-16 flex items-center justify-between px-6 border-b border-[#222] bg-[#141414]/80 backdrop-blur-md sticky top-0 z-30">
			<div class="flex items-center gap-4">
				<button class="p-2 border border-[#333] rounded-xl bg-[#1a1a1a] text-[#888] hover:bg-[#DBBA87] hover:text-black hover:border-[#DBBA87] transition-all shadow-sm" onclick={() => isSidebarOpen = !isSidebarOpen} aria-label="Toggle Sidebar"><Menu size={20} /></button>
				<button class="flex flex-col group cursor-pointer text-left bg-transparent border-none p-0" onclick={() => openRename(-1)} aria-label="Rename conversation">
					<div class="flex items-center gap-2"><span class="font-serif font-bold text-[#e5e5e5] text-lg leading-none group-hover:text-[#DBBA87] transition-colors">{currentTitle}</span><Edit2 size={12} class="opacity-0 group-hover:opacity-50 text-[#888] transition-opacity"/></div>
					<div class="flex items-center gap-1.5 mt-0.5"><div class="w-1.5 h-1.5 rounded-full bg-green-500 shadow-[0_0_5px_rgba(34,197,94,0.5)]"></div><span class="font-serif italic text-[10px] font-bold tracking-widest text-[#666]">DEEPSEEK V3</span></div>
				</button>
			</div>
			<button onclick={clearChat} class="text-[#666] hover:text-red-400 transition-colors"><Trash2 size={18}/></button>
		</header>

		<section bind:this={scrollContainer} class="flex-1 overflow-y-auto p-4 md:p-8 space-y-8 scroll-smooth">
    <div class="max-w-4xl mx-auto space-y-8">
       <!-- 找到这一行 -->
<!-- 修改这一行 -->
{#each chatHistory as msg, i (msg.content + i)}
    <ChatItem 
        {msg} 
        index={i} 
        isLast={i === chatHistory.length - 1} 
        onEdit={handleEdit} 
        onRegenerate={handleRegenerate}
        onSwitchVersion={handleSwitchVersion}
        onContinue={handleContinue}
    />
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
</style>